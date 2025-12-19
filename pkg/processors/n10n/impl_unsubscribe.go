/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"net/http"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/pipeline"
)

func unsubscribePipeline(requestCtx context.Context, p *implIN10NProc) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(requestCtx, "Unsubscribe Processor",
		pipeline.WireFunc("validateToken", p.validateToken),
		pipeline.WireFunc("denyBody", denyBody),
		pipeline.WireFunc("unsubscribe", p.unsubscribe),
		pipeline.WireFunc("reply204NoContent", reply204NoContent),
	)
}

func (p *implIN10NProc) unsubscribe(ctx context.Context, n10nWP *n10nWorkpiece) (err error) {
	projectionKey := in10n.ProjectionKey{
		App:        n10nWP.appQName,
		Projection: n10nWP.entityFromURL,
		WS:         n10nWP.wsidFromURL,
	}
	return p.n10nBroker.Unsubscribe(n10nWP.channelID, projectionKey)
}

func reply204NoContent(ctx context.Context, n10nWP *n10nWorkpiece) (err error) {
	return n10nWP.responder.Respond(bus.ResponseMeta{StatusCode: http.StatusNoContent}, nil)
}
