/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"time"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/pipeline"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type ResultFieldsOperator struct {
	pipeline.AsyncNOOP
	elements     []IElement
	rootSchema   coreutils.SchemaFields
	schemasCache *schemasCache
	metrics      IMetrics
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
			err = o.fillRow(ctx, outputRow, element, object, o.rootSchema)
			if err != nil {
				return work, err
			}
			continue
		}
		var findElements func(parent istructs.IElement, pathEntries []string, pathEntryIndex int)
		findElements = func(parent istructs.IElement, pathEntries []string, pathEntryIndex int) {
			parent.Elements(pathEntries[pathEntryIndex], func(el istructs.IElement) {
				if pathEntryIndex == len(pathEntries)-1 {
					err = o.fillRow(ctx, outputRow, element, el, o.schemasCache.get(el.QName()))
					if err != nil {
						return
					}
				} else {
					findElements(el, pathEntries, pathEntryIndex+1)
				}
			})
		}
		findElements(object, element.Path().AsArray(), 0)
	}
	return work, err
}

func (o ResultFieldsOperator) fillRow(ctx context.Context, outputRow IOutputRow, element IElement, object istructs.IObject, schemaFields coreutils.SchemaFields) (err error) {
	row := element.NewOutputRow()
	for _, field := range element.ResultFields() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		value := coreutils.ReadByKind(field.Field(), schemaFields[field.Field()], object)
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
