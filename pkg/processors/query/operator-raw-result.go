/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
)

type RawResultOperator struct {
	pipeline.AsyncNOOP
	metrics IMetrics
}

func (o RawResultOperator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecFieldsSeconds, time.Since(begin).Seconds())
	}()
	topOutputRow := work.(IWorkpiece).OutputRow()
	object := work.(IWorkpiece).Object()
	row := &outputRow{
		keyToIdx: map[string]int{processors.Field_RawObject_Body: 0},
		values:   make([]interface{}, 1),
	}
	row.Set(processors.Field_RawObject_Body, object.AsString(processors.Field_RawObject_Body))
	topOutputRow.Set(rootDocument, []IOutputRow{row})
	return work, err
}
