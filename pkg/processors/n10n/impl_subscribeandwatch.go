/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

func subscribeAndWatchPipeline(requestCtx context.Context, p *implIN10NProc) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(requestCtx, "Subscribe and Watch Processor",
		pipeline.WireFunc("validateToken", p.validateToken),
		pipeline.WireFunc("getSubjectLogin", p.getSubjectLogin),
		pipeline.WireFunc("parseSubscribeAndWatchArgs", parseSubscribeAndWatchArgs),
		pipeline.WireFunc("newChannel", p.newChannel),
		pipeline.WireFunc("initResponse", initResponse),
		pipeline.WireFunc("sendChannelIDSSEEvent", sendChannelIDSSEEvent),
		pipeline.WireFunc("subscribe", p.subscribe),
		pipeline.WireFunc("go WatchChannel", p.watchChannel),
	)
}

func (p *implIN10NProc) validateToken(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	appTokens := p.appTokensFactory.New(n10nWP.appQName)
	_, err = appTokens.ValidateToken(n10nWP.token, &n10nWP.principalPayload)

	// [~server.n10n/err.routerCreateChannelInvalidToken~impl]
	// [~server.n10n/err.routerAddSubscriptionInvalidToken~impl]
	// [~server.n10n/err.routerUnsubscribeInvalidToken~impl]
	return coreutils.WrapSysError(err, http.StatusUnauthorized)
}

func (p *implIN10NProc) getSubjectLogin(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	n10nWP.subjectLogin = istructs.SubjectLogin(n10nWP.principalPayload.Login)
	return nil
}

func parseSubscribeAndWatchArgs(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	subscribeAndWatchArgs := n10nArgs{}
	if err := coreutils.JSONUnmarshalDisallowUnknownFields(n10nWP.body, &subscribeAndWatchArgs); err != nil {
		return fmt.Errorf("failed to unmarshal request body: %w", err)
	}
	if subscribeAndWatchArgs.ExpiresInSeconds == 0 {
		subscribeAndWatchArgs.ExpiresInSeconds = defaultN10NExpiresInSeconds
	} else if subscribeAndWatchArgs.ExpiresInSeconds < 0 {
		return fmt.Errorf("invalid expiresIn value %d", subscribeAndWatchArgs.ExpiresInSeconds)
	}
	n10nWP.expiresIn = time.Duration(subscribeAndWatchArgs.ExpiresInSeconds) * time.Second
	if len(subscribeAndWatchArgs.Subscriptions) == 0 {
		return errors.New("no subscriptions provided")
	}
	for i, subscr := range subscribeAndWatchArgs.Subscriptions {
		if len(subscr.Entity) == 0 || len(subscr.WSIDNumber.String()) == 0 {
			return fmt.Errorf("subscriptions[%d]: entity and/or wsid is not provided", i)
		}
		wsid, err := coreutils.ClarifyJSONWSID(subscr.WSIDNumber)
		if err != nil {
			return fmt.Errorf("subscriptions[%d]: failed to parse wsid %s: %w", i, subscr.WSIDNumber, err)
		}
		entity, err := appdef.ParseQName(subscr.Entity)
		if err != nil {
			return fmt.Errorf("subscriptions[%d]: failed to parse entity %s as a QName: %w", i, subscr.Entity, err)
		}
		n10nWP.subscriptions = append(n10nWP.subscriptions, subscription{
			entity: entity,
			wsid:   wsid,
		})
	}
	return nil
}

func (p *implIN10NProc) newChannel(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	n10nWP.channelID, err = p.n10nBroker.NewChannel(n10nWP.subjectLogin, n10nWP.expiresIn)
	return err
}

func initResponse(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	n10nWP.responseWriter = n10nWP.responder.StreamEvents()
	return nil
}

func sendChannelIDSSEEvent(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	return n10nWP.responseWriter.Write(fmt.Sprintf("event: channelId\ndata: %s\n\n", n10nWP.channelID))
}

func (p *implIN10NProc) subscribe(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	for _, sub := range n10nWP.subscriptions {
		projectionKey := in10n.ProjectionKey{
			App:        n10nWP.appQName,
			Projection: sub.entity,
			WS:         sub.wsid,
		}
		if err = p.n10nBroker.Subscribe(n10nWP.channelID, projectionKey); err != nil {
			return fmt.Errorf("subscribe failed: %w", err)
		}
		n10nWP.subscribedProjectionKeys = append(n10nWP.subscribedProjectionKeys, projectionKey)
	}
	return nil
}

func (p *implIN10NProc) watchChannel(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	p.goroutinesWG.Add(1)
	watchChannelCtx, cancel := context.WithCancel(n10nWP.requestCtx)
	go func() {
		defer p.goroutinesWG.Done()
		// unsubscribe and channel cleanup is done within WatchChannel
		p.n10nBroker.WatchChannel(watchChannelCtx, n10nWP.channelID, func(projection in10n.ProjectionKey, offset istructs.Offset) {
			sseMessage := fmt.Sprintf("event: %s\ndata: %d\n\n", projection.ToJSON(), offset)
			if err := n10nWP.responseWriter.Write(sseMessage); err != nil {
				// could happen if e.g. router stopped to listen for bus
				logger.Error("failed to send sse message:", sseMessage)
				// force WatchChannel to exit
				cancel()
			}
		})
		n10nWP.responseWriter.Close(nil)
	}()
	return nil
}
