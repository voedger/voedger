/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/pipeline"
)

type SendToBusOperator struct {
	pipeline.AsyncNOOP
	sender    bus.IStreamingResponseSender
	responder bus.IResponder
	metrics   IMetrics
	errCh     chan<- error
}

func (o *SendToBusOperator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecSendSeconds, time.Since(begin).Seconds())
	}()
	if o.sender == nil {
		o.sender = o.responder.BeginStreamingResponse(http.StatusOK)
	}
	return work, o.sender.Send(work.(rowsWorkpiece).OutputRow().Values())
}

func (o *SendToBusOperator) OnError(_ context.Context, err error) {
	select {
	case o.errCh <- err:
	default:
		logger.Error("failed to send error from rowsProcessor to QP: " + err.Error())
	}
}
