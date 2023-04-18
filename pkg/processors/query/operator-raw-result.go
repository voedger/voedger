/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/pipeline"
)

type RawResultOperator struct {
	pipeline.AsyncNOOP
	metrics IMetrics
}

func (o RawResultOperator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(execFieldsSeconds, time.Since(begin).Seconds())
	}()
	topOutputRow := work.(IWorkpiece).OutputRow()
	object := work.(IWorkpiece).Object()
	row := &outputRow{
		keyToIdx: map[string]int{Field_JSONSchemaBody: 0},
		values:   make([]interface{}, 1),
	}
	row.Set(Field_JSONSchemaBody, object.AsString(Field_JSONSchemaBody))
	topOutputRow.Set(rootDocument, []IOutputRow{row})
	return work, err
}
