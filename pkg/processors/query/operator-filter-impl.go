/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/pipeline"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type FilterOperator struct {
	pipeline.AsyncNOOP
	filters    []IFilter
	rootFields coreutils.FieldsDef
	metrics    IMetrics
}

func (o FilterOperator) DoAsync(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(execFilterSeconds, time.Since(begin).Seconds())
	}()
	outputRow := work.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0]
	mergedFields := make(map[string]appdef.DataKind)
	for n, k := range o.rootFields {
		mergedFields[n] = k
	}
	for n, k := range work.(IWorkpiece).EnrichedRootFields() {
		mergedFields[n] = k
	}
	for _, filter := range o.filters {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		match, err := filter.IsMatch(mergedFields, outputRow)
		if err != nil {
			return nil, err
		}
		if !match {
			work.Release()
			return nil, nil
		}
	}
	return work, nil
}
