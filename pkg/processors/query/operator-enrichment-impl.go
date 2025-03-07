/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/sys"
)

type EnrichmentOperator struct {
	pipeline.AsyncNOOP
	state      istructs.IState
	elements   []IElement
	fieldsDefs *fieldsDefs
	metrics    IMetrics
}

func (o *EnrichmentOperator) DoAsync(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecEnrichSeconds, time.Since(begin).Seconds())
	}()
	outputRow := work.(IWorkpiece).OutputRow()
	for _, element := range o.elements {
		rows := outputRow.Value(element.Path().Name()).([]IOutputRow)
		for i := range rows {
			for _, field := range element.RefFields() {
				if ctx.Err() != nil {
					return work, ctx.Err()
				}

				kb, err := o.state.KeyBuilder(sys.Storage_Record, appdef.NullQName)
				if err != nil {
					return work, err
				}
				kb.PutRecordID(sys.Storage_Record_Field_ID, rows[i].Value(field.Key()).(istructs.RecordID))

				sv, err := o.state.MustExist(kb)
				if err != nil {
					return work, err
				}
				record := sv.(istructs.IStateRecordValue).AsRecord()

				recFields := o.fieldsDefs.get(record.QName())
				refFieldKind, ok := recFields[field.RefField()]
				if !ok {
					return work, coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("ref field %s references to table %s that does not contain field %s", field.Field(), record.QName(), field.RefField()))
				}
				value := coreutils.ReadByKind(field.RefField(), refFieldKind, record)
				if element.Path().IsRoot() {
					work.(IWorkpiece).PutEnrichedRootFieldKind(field.Key(), recFields[field.RefField()])
				}
				rows[i].Set(field.Key(), value)
			}
		}
	}
	return work, err
}
