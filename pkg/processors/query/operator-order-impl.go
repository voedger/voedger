/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/voedger/voedger/pkg/pipeline"
)

type OrderOperator struct {
	pipeline.AsyncNOOP
	orderBys []IOrderBy
	rows     []IOutputRow
	metrics  IMetrics
}

func newOrderOperator(orderBys []IOrderBy, metrics IMetrics) pipeline.IAsyncOperator {
	return &OrderOperator{
		orderBys: orderBys,
		rows:     make([]IOutputRow, 0),
		metrics:  metrics,
	}
}

func (o *OrderOperator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecOrderSeconds, time.Since(begin).Seconds())
	}()
	o.rows = append(o.rows, work.(IWorkpiece).OutputRow())
	work.Release()
	return nil, nil
}

func (o *OrderOperator) Flush(callback pipeline.OpFuncFlush) (err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecOrderSeconds, time.Since(begin).Seconds())
	}()
	slices.SortFunc(o.rows, func(a, b IOutputRow) int {
		for _, orderBy := range o.orderBys {
			o1 := o.value(a, orderBy.Field())
			o2 := o.value(b, orderBy.Field())
			var c int
			switch v := o1.(type) {
			case int32:
				c = cmp.Compare(v, o2.(int32))
			case int64:
				c = cmp.Compare(v, o2.(int64))
			case float32:
				c = cmp.Compare(v, o2.(float32))
			case float64:
				c = cmp.Compare(v, o2.(float64))
			case string:
				c = cmp.Compare(v, o2.(string))
			default:
				err = fmt.Errorf("order by '%s' is impossible: %w", orderBy.Field(), ErrWrongType)
				return 0
			}
			if c != 0 {
				if orderBy.IsDesc() {
					return -c
				}
				return c
			}
		}
		return 0
	})
	if err == nil {
		for _, row := range o.rows {
			callback(rowsWorkpiece{outputRow: row})
		}
	}
	return err
}

func (o *OrderOperator) value(row IOutputRow, field string) interface{} {
	return row.Value(rootDocument).([]IOutputRow)[0].Value(field)
}
