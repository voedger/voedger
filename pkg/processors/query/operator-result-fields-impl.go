/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type ResultFieldsOperator struct {
	pipeline.AsyncNOOP
	elements   []IElement
	rootFields FieldsKinds
	fieldsDefs *fieldsDefs
	metrics    IMetrics
}

func (o ResultFieldsOperator) DoAsync(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(execFieldsSeconds, time.Since(begin).Seconds())
	}()
	outputRow := work.(IWorkpiece).OutputRow()
	object := work.(IWorkpiece).Object()
	for _, element := range o.elements {
		outputRow.Set(element.Path().Name(), make([]IOutputRow, 0))
		if element.Path().IsRoot() {
			err = o.fillRow(ctx, outputRow, element, object, o.rootFields)
			if err != nil {
				return work, err
			}
			continue
		}
		var findElements func(parent istructs.IObject, pathEntries []string, pathEntryIndex int)
		findElements = func(parent istructs.IObject, pathEntries []string, pathEntryIndex int) {
			for c := range parent.Children(pathEntries[pathEntryIndex]) {
				if pathEntryIndex == len(pathEntries)-1 {
					err = o.fillRow(ctx, outputRow, element, c, o.fieldsDefs.get(c.QName()))
					if err != nil {
						return
					}
				} else {
					findElements(c, pathEntries, pathEntryIndex+1)
				}
			}
		}
		findElements(object, element.Path().AsArray(), 0)
	}
	return work, err
}

func (o ResultFieldsOperator) fillRow(ctx context.Context, outputRow IOutputRow, element IElement, object istructs.IObject, fk FieldsKinds) (err error) {
	row := element.NewOutputRow()
	for _, field := range element.ResultFields() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		value := coreutils.ReadByKind(field.Field(), fk[field.Field()], object)
		row.Set(field.Field(), value)
	}
	for _, field := range element.RefFields() {
		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}
		row.Set(field.Key(), object.AsRecordID(field.Field()))
	}
	outputRow.Set(element.Path().Name(), append(outputRow.Value(element.Path().Name()).([]IOutputRow), row))
	return nil
}
