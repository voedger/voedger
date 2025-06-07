/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"fmt"
	"sort"
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

func (o OrderOperator) Flush(callback pipeline.OpFuncFlush) (err error) {
	begin := time.Now()
	defer func() {
		o.metrics.Increase(Metric_ExecOrderSeconds, time.Since(begin).Seconds())
	}()
	sort.Slice(o.rows, func(i, j int) bool {
		for _, orderBy := range o.orderBys {
			o1 := o.value(i, orderBy.Field())
			o2 := o.value(j, orderBy.Field())
			if o1 == o2 {
				continue
			}
			switch v := o1.(type) {
			case int32:
				return compareInt32(v, o2.(int32), orderBy.IsDesc())
			case int64:
				return compareInt64(v, o2.(int64), orderBy.IsDesc())
			case float32:
				return compareFloat32(v, o2.(float32), orderBy.IsDesc())
			case float64:
				return compareFloat64(v, o2.(float64), orderBy.IsDesc())
			case string:
				return compareString(v, o2.(string), orderBy.IsDesc())
			default:
				err = fmt.Errorf("order by '%s' is impossible: %w", orderBy.Field(), ErrWrongType)
			}
		}
		return false
	})
	if err == nil {
		for _, row := range o.rows {
			callback(rowsWorkpiece{outputRow: row})
		}
	}
	return err
}

func (o OrderOperator) value(i int, field string) interface{} {
	return o.rows[i].Value(rootDocument).([]IOutputRow)[0].Value(field)
}

func compareInt32(o1, o2 int32, desc bool) bool {
	if desc {
		return o1 > o2
	}
	return o1 < o2
}

func compareInt64(o1, o2 int64, desc bool) bool {
	if desc {
		return o1 > o2
	}
	return o1 < o2
}

func compareFloat32(o1, o2 float32, desc bool) bool {
	if desc {
		return o1 > o2
	}
	return o1 < o2
}

func compareFloat64(o1, o2 float64, desc bool) bool {
	if desc {
		return o1 > o2
	}
	return o1 < o2
}

func compareString(o1, o2 string, desc bool) bool {
	if desc {
		return o1 > o2
	}
	return o1 < o2
}
