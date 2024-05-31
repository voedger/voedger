/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
*/

package imetrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	metrics := Provide()

	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app1, 0.0002)
	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app1, 0.00015)
	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app2, 1)
	metrics.Increase("somecounter_total", "host1", 7)

	collection := make(map[string]bool)
	_ = metrics.List(func(metric IMetric, metricValue float64) (err error) {
		collection[string(ToPrometheus(metric, metricValue))] = true
		return err
	})

	require.Len(collection, 3)
	require.True(collection["somecounter_total{app=\"test1/app1\",vvm=\"host1\"} 0.00035\n"])
	require.True(collection["somecounter_total{app=\"test1/app2\",vvm=\"host1\"} 1\n"])
	require.True(collection["somecounter_total{vvm=\"host1\"} 7\n"])
}

func TestMetrics_List(t *testing.T) {
	require := require.New(t)

	testErr := errors.New("boom")
	times := 0

	metrics := Provide()

	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app1, 0.0002)
	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app1, 0.00015)
	metrics.IncreaseApp("somecounter_total", "host1", istructs.AppQName_test1_app2, 1)
	metrics.Increase("somecounter_total", "host1", 7)

	err := metrics.List(func(metric IMetric, metricValue float64) (err error) {
		times++
		return testErr
	})

	require.ErrorIs(err, testErr)
	require.Equal(1, times)
}

func TestToPrometheus(t *testing.T) {
	tests := []struct {
		name  string
		app   appdef.AppQName
		vvm   string
		value float64
		want  string
	}{
		{
			name:  "Full",
			app:   istructs.AppQName_test1_app1,
			vvm:   "host",
			value: 164759,
			want:  "something_total{app=\"test1/app1\",vvm=\"host\"} 164759\n",
		},
		{
			name:  "Without app",
			app:   appdef.NullAppQName,
			vvm:   "host",
			value: 164759,
			want:  "something_total{vvm=\"host\"} 164759\n",
		},
		{
			name:  "Without vvm",
			app:   istructs.AppQName_test1_app1,
			vvm:   "",
			value: 164759,
			want:  "something_total{app=\"test1/app1\"} 164759\n",
		},
		{
			name:  "Big value",
			app:   istructs.AppQName_test2_app1,
			vvm:   "host",
			value: 123456789000.01,
			want:  "something_total{app=\"test2/app1\",vvm=\"host\"} 123456789000.01\n",
		},
		{
			name:  "Small value",
			app:   istructs.AppQName_test2_app1,
			vvm:   "host",
			value: 0.00000123456,
			want:  "something_total{app=\"test2/app1\",vvm=\"host\"} 0.00000123456\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &metric{
				name: "something_total",
				app:  test.app,
				vvm:  test.vvm,
			}

			require.Equal(t, test.want, string(ToPrometheus(m, test.value)))
		})
	}
}
