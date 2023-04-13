/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/pipeline"
	"github.com/untillpro/voedger/pkg/state"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type EnrichmentOperator struct {
	pipeline.AsyncNOOP
	state        istructs.IState
	elements     []IElement
	schemasCache *schemasCache
	metrics      IMetrics
}

func (o *EnrichmentOperator) DoAsync(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(execEnrichSeconds, time.Since(begin).Seconds())
	}()
	outputRow := work.(IWorkpiece).OutputRow()
	for _, element := range o.elements {
		rows := outputRow.Value(element.Path().Name()).([]IOutputRow)
		for i := range rows {
			for _, field := range element.RefFields() {
				if ctx.Err() != nil {
					return work, ctx.Err()
				}

				kb, err := o.state.KeyBuilder(state.RecordsStorage, istructs.NullQName)
				if err != nil {
					return work, err
				}
				kb.PutRecordID(state.Field_ID, rows[i].Value(field.Key()).(istructs.RecordID))

				sv, err := o.state.MustExist(kb)
				if err != nil {
					return work, err
				}
				record := sv.AsRecord("")

				schemaFields := o.schemasCache.get(record.QName())
				value := coreutils.ReadByKind(field.RefField(), schemaFields[field.RefField()], record)
				if element.Path().IsRoot() {
					work.(IWorkpiece).PutEnrichedRootSchemaField(field.Key(), schemaFields[field.RefField()])
				}
				rows[i].Set(field.Key(), value)
			}
		}
	}
	return work, err
}
