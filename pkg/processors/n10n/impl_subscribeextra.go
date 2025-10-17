/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"errors"
	"net/http"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/pipeline"
)

func subscribeExtraPipeline(requestCtx context.Context, p *implIN10NProc) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(requestCtx, "Subscribe on Extra View Processor",
		pipeline.WireFunc("validateToken", p.validateToken),
		pipeline.WireFunc("denyBody", denyBody),
		pipeline.WireFunc("getAppStructs", p.getAppStructs),
		pipeline.WireFunc("addProjectionKeyFromURL", addProjectionKeyFromURL),
		pipeline.WireFunc("authnzEntities", p.authnzEntities),
		pipeline.WireFunc("subscribe", p.subscribe),
		pipeline.WireFunc("replyOK", p.replyOK),
	)
}

func denyBody(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	if len(n10nWP.body) > 0 {
		return errors.New("unexpected body")
	}
	return nil
}

func addProjectionKeyFromURL(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	n10nWP.subscriptions = append(n10nWP.subscriptions, subscription{
		entity: n10nWP.entityFromURL,
		wsid:   n10nWP.wsidFromURL,
	})
	return nil
}

func (p *implIN10NProc) replyOK(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	n10nWP := work.(*n10nWorkpiece)
	return n10nWP.responder.Respond(bus.ResponseMeta{StatusCode: http.StatusOK}, nil)
}
