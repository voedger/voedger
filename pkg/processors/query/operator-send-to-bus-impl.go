/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/pipeline"
)

type SendToBusOperator struct {
	pipeline.AsyncNOOP
	rs          IResultSenderClosable
	initialized bool
	metrics     IMetrics
}

func (o *SendToBusOperator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(execSendSeconds, time.Since(begin).Seconds())
	}()
	if !o.initialized {
		//TODO what to set into sectionType, path?
		o.rs.StartArraySection("", nil)
		o.initialized = true
	}
	return work, o.rs.SendElement("", work.(rowsWorkpiece).OutputRow().Values())
}

func (o *SendToBusOperator) OnError(_ context.Context, err error) {
	o.rs.Close(err)
}
