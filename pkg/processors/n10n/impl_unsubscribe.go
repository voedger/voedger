/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"net/http"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/pipeline"
)

func unsubscribePipeline(requestCtx context.Context, p *implIN10NProc) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(requestCtx, "Unsubscribe Processor",
		pipeline.WireFunc("validateToken", p.validateToken),
		pipeline.WireFunc("denyBody", denyBody),
		pipeline.WireFunc("unsubscribe", p.unsubscribe),
		pipeline.WireFunc("logUnsubscribeSuccess", logUnsubscribeSuccess),
		pipeline.WireFunc("reply204NoContent", reply204NoContent),
	)
}

func (p *implIN10NProc) unsubscribe(ctx context.Context, n10nWP *n10nWorkpiece) (err error) {
	projectionKey := in10n.ProjectionKey{
		App:        n10nWP.appQName,
		Projection: n10nWP.entityFromURL,
		WS:         n10nWP.wsidFromURL,
	}
	if err = p.n10nBroker.Unsubscribe(n10nWP.channelID, projectionKey); err != nil {
		logger.ErrorCtx(n10nProjectionLogCtx(n10nWP.logCtx, projectionKey), "n10n.unsubscribe.error", err)
		return err
	}
	n10nWP.subscribedProjectionKeys = append(n10nWP.subscribedProjectionKeys, projectionKey)
	return nil
}

func logUnsubscribeSuccess(ctx context.Context, n10nWP *n10nWorkpiece) (err error) {
	if logger.IsVerbose() {
		for _, pk := range n10nWP.subscribedProjectionKeys {
			logger.VerboseCtx(n10nProjectionLogCtx(n10nWP.logCtx, pk), "n10n.unsubscribe.success")
		}
	}
	return nil
}

func reply204NoContent(ctx context.Context, n10nWP *n10nWorkpiece) (err error) {
	return n10nWP.responder.Respond(bus.ResponseMeta{StatusCode: http.StatusNoContent}, nil)
}
