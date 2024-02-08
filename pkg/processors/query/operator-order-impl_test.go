/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/pipeline"
)

func TestOrderOperator_Flush(t *testing.T) {
	work := func(id int64, name string, departmentNumber int64, weight float64) pipeline.IWorkpiece {
		return rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values: []interface{}{
					[]IOutputRow{
						&outputRow{
							keyToIdx: map[string]int{
								"id":                0,
								"name":              1,
								"department_number": 2,
								"weight":            3,
							},
							values: []interface{}{id, name, departmentNumber, weight},
						},
					},
				},
			},
		}
	}
	id := func(work pipeline.IWorkpiece) int64 {
		return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[0].(int64)
	}
	name := func(work pipeline.IWorkpiece) string {
		return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[1].(string)
	}
	departmentNumber := func(work pipeline.IWorkpiece) int64 {
		return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[2].(int64)
	}
	weight := func(work pipeline.IWorkpiece) float64 {
		return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[3].(float64)
	}
	t.Run("Should order by one int64 field asc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "id",
				desc:  false,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(1), id(works[0]))
		require.Equal("Cola", name(works[0]))
		require.Equal(int64(5), id(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(int64(42), id(works[2]))
		require.Equal("Sprite", name(works[2]))
	})
	t.Run("Should order by one int64 field desc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "id",
				desc:  true,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(42), id(works[0]))
		require.Equal("Sprite", name(works[0]))
		require.Equal(int64(5), id(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(int64(1), id(works[2]))
		require.Equal("Cola", name(works[2]))
	})
	t.Run("Should order by one string field asc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "name",
				desc:  false,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(1), id(works[0]))
		require.Equal("Cola", name(works[0]))
		require.Equal(int64(5), id(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(int64(42), id(works[2]))
		require.Equal("Sprite", name(works[2]))
	})
	t.Run("Should order by one string field desc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "name",
				desc:  true,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(42), id(works[0]))
		require.Equal("Sprite", name(works[0]))
		require.Equal(int64(5), id(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(int64(1), id(works[2]))
		require.Equal("Cola", name(works[2]))
	})
	t.Run("Should order by one float64 field asc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "weight",
				desc:  false,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(1.15, weight(works[0]))
		require.Equal("Cola", name(works[0]))
		require.Equal(1.75, weight(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(2.0, weight(works[2]))
		require.Equal("Sprite", name(works[2]))
	})
	t.Run("Should order by one float64 field desc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "weight",
				desc:  true,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Cola", 100, 1.15))
		_, _ = operator.DoAsync(context.Background(), work(42, "Sprite", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(5, "Pepsi", 100, 1.75))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(2.0, weight(works[0]))
		require.Equal("Sprite", name(works[0]))
		require.Equal(1.75, weight(works[1]))
		require.Equal("Pepsi", name(works[1]))
		require.Equal(1.15, weight(works[2]))
		require.Equal("Cola", name(works[2]))
	})
	t.Run("Should order by two fields asc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "department_number",
				desc:  false,
			},
			orderBy{
				field: "name",
				desc:  false,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Xenta", 100, 1.45))
		_, _ = operator.DoAsync(context.Background(), work(2, "Amaretto", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(3, "Vodka", 200, 2.13))
		_, _ = operator.DoAsync(context.Background(), work(4, "Sherry", 200, 1.7))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(100), departmentNumber(works[0]))
		require.Equal("Amaretto", name(works[0]))
		require.Equal(int64(100), departmentNumber(works[1]))
		require.Equal("Xenta", name(works[1]))
		require.Equal(int64(200), departmentNumber(works[2]))
		require.Equal("Sherry", name(works[2]))
		require.Equal(int64(200), departmentNumber(works[3]))
		require.Equal("Vodka", name(works[3]))
	})
	t.Run("Should order by two fields desc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "department_number",
				desc:  true,
			},
			orderBy{
				field: "name",
				desc:  true,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Xenta", 100, 1.45))
		_, _ = operator.DoAsync(context.Background(), work(2, "Amaretto", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(3, "Vodka", 200, 2.13))
		_, _ = operator.DoAsync(context.Background(), work(4, "Sherry", 200, 1.7))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(200), departmentNumber(works[0]))
		require.Equal("Vodka", name(works[0]))
		require.Equal(int64(200), departmentNumber(works[1]))
		require.Equal("Sherry", name(works[1]))
		require.Equal(int64(100), departmentNumber(works[2]))
		require.Equal("Xenta", name(works[2]))
		require.Equal(int64(100), departmentNumber(works[3]))
		require.Equal("Amaretto", name(works[3]))
	})
	t.Run("Should order by two fields first field is asc second filed is desc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "department_number",
				desc:  false,
			},
			orderBy{
				field: "name",
				desc:  true,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Xenta", 100, 1.45))
		_, _ = operator.DoAsync(context.Background(), work(2, "Amaretto", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(3, "Vodka", 200, 2.13))
		_, _ = operator.DoAsync(context.Background(), work(4, "Sherry", 200, 1.7))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(100), departmentNumber(works[0]))
		require.Equal("Xenta", name(works[0]))
		require.Equal(int64(100), departmentNumber(works[1]))
		require.Equal("Amaretto", name(works[1]))
		require.Equal(int64(200), departmentNumber(works[2]))
		require.Equal("Vodka", name(works[2]))
		require.Equal(int64(200), departmentNumber(works[3]))
		require.Equal("Sherry", name(works[3]))
	})
	t.Run("Should order by two fields first field is desc second filed is asc", func(t *testing.T) {
		require := require.New(t)
		orders := []IOrderBy{
			orderBy{
				field: "department_number",
				desc:  true,
			},
			orderBy{
				field: "name",
				desc:  false,
			}}
		operator := newOrderOperator(orders, &testMetrics{})
		works := make([]pipeline.IWorkpiece, 0)

		_, _ = operator.DoAsync(context.Background(), work(1, "Xenta", 100, 1.45))
		_, _ = operator.DoAsync(context.Background(), work(2, "Amaretto", 100, 2.0))
		_, _ = operator.DoAsync(context.Background(), work(3, "Vodka", 200, 2.13))
		_, _ = operator.DoAsync(context.Background(), work(4, "Sherry", 200, 1.7))

		_ = operator.Flush(func(work pipeline.IWorkpiece) {
			works = append(works, work)
		})

		require.Equal(int64(200), departmentNumber(works[0]))
		require.Equal("Sherry", name(works[0]))
		require.Equal(int64(200), departmentNumber(works[1]))
		require.Equal("Vodka", name(works[1]))
		require.Equal(int64(100), departmentNumber(works[2]))
		require.Equal("Amaretto", name(works[2]))
		require.Equal(int64(100), departmentNumber(works[3]))
		require.Equal("Xenta", name(works[3]))
	})
	t.Run("Should return data type unspecified error", func(t *testing.T) {
		require := require.New(t)
		work := func(flag bool) pipeline.IWorkpiece {
			return rowsWorkpiece{
				outputRow: &outputRow{
					keyToIdx: map[string]int{rootDocument: 0},
					values: []interface{}{
						[]IOutputRow{
							&outputRow{
								keyToIdx: map[string]int{"flag": 0},
								values:   []interface{}{flag},
							},
						},
					},
				},
			}
		}
		orders := []IOrderBy{
			orderBy{
				field: "flag",
				desc:  false,
			},
		}
		operator := newOrderOperator(orders, &testMetrics{})

		_, _ = operator.DoAsync(context.Background(), work(true))
		_, _ = operator.DoAsync(context.Background(), work(false))

		err := operator.Flush(func(work pipeline.IWorkpiece) {
			t.Fatal("must not be call")
		})

		require.ErrorIs(err, ErrWrongType)
	})
	t.Run("Should order by int32 field", func(t *testing.T) {
		work := func(x, y int32) pipeline.IWorkpiece {
			return rowsWorkpiece{outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values: []interface{}{
					[]IOutputRow{&outputRow{
						keyToIdx: map[string]int{
							"x": 0,
							"y": 1,
						},
						values: []interface{}{x, y},
					}},
				},
			}}
		}
		x := func(work pipeline.IWorkpiece) int32 {
			return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[0].(int32)
		}
		y := func(work pipeline.IWorkpiece) int32 {
			return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[1].(int32)
		}
		t.Run("Asc order", func(t *testing.T) {
			require := require.New(t)
			orders := []IOrderBy{
				orderBy{
					field: "x",
					desc:  false,
				}}
			operator := newOrderOperator(orders, &testMetrics{})
			works := make([]pipeline.IWorkpiece, 0)

			_, _ = operator.DoAsync(context.Background(), work(0, 1))
			_, _ = operator.DoAsync(context.Background(), work(2, 3))

			_ = operator.Flush(func(work pipeline.IWorkpiece) {
				works = append(works, work)
			})

			require.Equal(int32(0), x(works[0]))
			require.Equal(int32(1), y(works[0]))
			require.Equal(int32(2), x(works[1]))
			require.Equal(int32(3), y(works[1]))
		})
		t.Run("Desc order", func(t *testing.T) {
			require := require.New(t)
			orders := []IOrderBy{
				orderBy{
					field: "x",
					desc:  true,
				}}
			operator := newOrderOperator(orders, &testMetrics{})
			works := make([]pipeline.IWorkpiece, 0)

			_, _ = operator.DoAsync(context.Background(), work(0, 1))
			_, _ = operator.DoAsync(context.Background(), work(2, 3))

			_ = operator.Flush(func(work pipeline.IWorkpiece) {
				works = append(works, work)
			})

			require.Equal(int32(2), x(works[0]))
			require.Equal(int32(3), y(works[0]))
			require.Equal(int32(0), x(works[1]))
			require.Equal(int32(1), y(works[1]))
		})
	})
	t.Run("Should order by float32 field", func(t *testing.T) {
		work := func(temperature float32) pipeline.IWorkpiece {
			return rowsWorkpiece{outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values: []interface{}{
					[]IOutputRow{&outputRow{
						keyToIdx: map[string]int{"temperature": 0},
						values:   []interface{}{temperature},
					}},
				},
			}}
		}
		temperature := func(work pipeline.IWorkpiece) float32 {
			return work.(rowsWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Values()[0].(float32)
		}
		t.Run("Asc order", func(t *testing.T) {
			require := require.New(t)
			orders := []IOrderBy{
				orderBy{
					field: "x",
					desc:  false,
				}}
			operator := newOrderOperator(orders, &testMetrics{})
			works := make([]pipeline.IWorkpiece, 0)

			_, _ = operator.DoAsync(context.Background(), work(22.5))
			_, _ = operator.DoAsync(context.Background(), work(-7.2))
			_, _ = operator.DoAsync(context.Background(), work(15.3))

			_ = operator.Flush(func(work pipeline.IWorkpiece) {
				works = append(works, work)
			})

			require.Equal(float32(-7.2), temperature(works[0]))
			require.Equal(float32(15.3), temperature(works[1]))
			require.Equal(float32(22.5), temperature(works[2]))
		})
		t.Run("Desc order", func(t *testing.T) {
			require := require.New(t)
			orders := []IOrderBy{
				orderBy{
					field: "x",
					desc:  true,
				}}
			operator := newOrderOperator(orders, &testMetrics{})
			works := make([]pipeline.IWorkpiece, 0)

			_, _ = operator.DoAsync(context.Background(), work(22.5))
			_, _ = operator.DoAsync(context.Background(), work(-7.2))
			_, _ = operator.DoAsync(context.Background(), work(15.3))

			_ = operator.Flush(func(work pipeline.IWorkpiece) {
				works = append(works, work)
			})

			require.Equal(float32(22.5), temperature(works[0]))
			require.Equal(float32(15.3), temperature(works[1]))
			require.Equal(float32(-7.2), temperature(works[2]))
		})
	})
}

func TestOrderOperator_DoAsync(t *testing.T) {
	//TODO
	require := require.New(t)
	release := false
	work := testWorkpiece{
		outputRow: &testOutputRow{},
		release: func() {
			release = true
		},
	}
	operator := newOrderOperator(nil, &testMetrics{})

	_, _ = operator.DoAsync(context.Background(), work)

	require.Len(operator.(*OrderOperator).rows, 1)
	require.True(release)
}
