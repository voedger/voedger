/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_splitID(t *testing.T) {
	tests := []struct {
		name    string
		id      uint64
		wantHi  uint64
		wantLow uint16
	}{
		{
			name:    "split null record must return zeros",
			id:      uint64(istructs.NullRecordID),
			wantHi:  0,
			wantLow: 0,
		},
		{
			name:    "split 4095 must return 0 and 4095",
			id:      4095,
			wantHi:  0,
			wantLow: 4095,
		},
		{
			name:    "split 4096 must return 1 and 0",
			id:      4096,
			wantHi:  1,
			wantLow: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHi, gotLow := crackID(tt.id)
			if gotHi != tt.wantHi {
				t.Errorf("splitID() got Hi = %v, want %v", gotHi, tt.wantHi)
			}
			if gotLow != tt.wantLow {
				t.Errorf("splitID() got Low = %v, want %v", gotLow, tt.wantLow)
			}
		})
	}
}

func Test_splitRecordID(t *testing.T) {
	tests := []struct {
		name   string
		id     istructs.RecordID
		wantPk []byte
		wantCc []byte
	}{
		{
			name:   "split null record must return {0} and {0}",
			id:     istructs.NullRecordID,
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 0},
			wantCc: []byte{0, 0},
		},
		{
			name:   "split 4095 must return {0} and {0x0F, 0xFF}",
			id:     istructs.RecordID(4095),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 0},
			wantCc: []byte{0x0F, 0xFF},
		},
		{
			name:   "split 4096 must return {1} and {0}",
			id:     istructs.RecordID(4096),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 1},
			wantCc: []byte{0, 0},
		},
		{
			name:   "split 4097 must return {1} and {1}",
			id:     istructs.RecordID(4097),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 1},
			wantCc: []byte{0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPk, gotCc := splitRecordID(tt.id)
			if !reflect.DeepEqual(gotPk, tt.wantPk) {
				t.Errorf("splitRecordID() gotPk = %v, want %v", gotPk, tt.wantPk)
			}
			if !reflect.DeepEqual(gotCc, tt.wantCc) {
				t.Errorf("splitRecordID() gotCc = %v, want %v", gotCc, tt.wantCc)
			}
		})
	}
}

func Test_splitCalcLogOffset(t *testing.T) {
	require := require.New(t)

	wg := sync.WaitGroup{}

	const basketCount int = 4
	testBaskets := func(startbasket int) {
		startOffs := istructs.Offset(startbasket * 4096)
		for ofs := startOffs; ofs < startOffs+istructs.Offset(4096*basketCount); ofs++ {
			pk, cc := splitLogOffset(ofs)
			ofs1 := calcLogOffset(pk, cc)
			require.Equal(ofs, ofs1)
		}
		wg.Done()
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go testBaskets(i * basketCount)
	}
	wg.Wait()
}

func Test_splitLogOffsetMonotonicIncrease(t *testing.T) {
	require := require.New(t)

	wg := sync.WaitGroup{}

	const basketCount int = 4
	testBaskets := func(startbasket int) {
		startOffs := istructs.Offset(startbasket * 4096)
		p, c := splitLogOffset(startOffs)
		for ofs := startOffs + 1; ofs < startOffs+istructs.Offset(4096*basketCount); ofs++ {
			pp, cc := splitLogOffset(ofs)
			if reflect.DeepEqual(pp, p) {
				require.Greater(string(cc), string(c))
			} else {
				require.Greater(string(pp), string(p))
				require.True(reflect.DeepEqual(c, []byte{0x0F, 0xFF}))
				require.True(reflect.DeepEqual(cc, []byte{0x00, 0x00}))
			}
			c = cc
			p = pp
		}

		wg.Done()
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go testBaskets(i * basketCount)
	}
	wg.Wait()
}

func TestElementFillAndGet(t *testing.T) {
	require := require.New(t)
	test := test()

	cfgs := test.AppConfigs
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	_, err := asp.AppStructs(test.appName)
	require.NoError(err)
	builder := NewIObjectBuilder(cfgs[istructs.AppQName_test1_app1], test.testCDoc)

	t.Run("basic", func(t *testing.T) {

		data := map[string]interface{}{
			"sys.ID":  float64(7),
			"int32":   float64(1),
			"int64":   float64(2),
			"float32": float64(3),
			"float64": float64(4),
			"bytes":   "BQY=", // []byte{5,6}
			"string":  "str",
			"QName":   "test.CDoc",
			"bool":    true,
			"record": []interface{}{
				map[string]interface{}{
					"sys.ID": float64(8),
					"int32":  float64(6),
				},
			},
		}
		cfg := cfgs[test.appName]
		require.NoError(FillElementFromJSON(data, cfg.Schemas.Schema(test.testCDoc), builder, cfg.app.Schemas()))
		o, err := builder.Build()
		require.NoError(err)

		require.Equal(istructs.RecordID(7), o.AsRecordID("sys.ID"))
		require.Equal(int32(1), o.AsInt32("int32"))
		require.Equal(int64(2), o.AsInt64("int64"))
		require.Equal(float32(3), o.AsFloat32("float32"))
		require.Equal(float64(4), o.AsFloat64("float64"))
		require.Equal([]byte{5, 6}, o.AsBytes("bytes"))
		require.Equal("str", o.AsString("string"))
		require.Equal(test.testCDoc, o.AsQName("QName"))
		require.True(o.AsBool("bool"))
		count := 0
		o.Elements("record", func(el istructs.IElement) {
			require.Equal(istructs.RecordID(8), el.AsRecordID("sys.ID"))
			require.Equal(int32(6), el.AsInt32("int32"))
			count++
		})
		require.Equal(1, count)
	})

	t.Run("type errors", func(t *testing.T) {
		cases := map[string]interface{}{
			"int32":   "str",
			"int64":   "str",
			"float32": "str",
			"float64": "str",
			"bytes":   float64(2),
			"string":  float64(3),
			"QName":   float64(4),
			"bool":    "str",
			"record": []interface{}{
				map[string]interface{}{"int32": "str"},
			},
		}

		cfg := cfgs[test.appName]
		for name, val := range cases {
			builder := NewIObjectBuilder(cfgs[istructs.AppQName_test1_app1], test.testCDoc)
			data := map[string]interface{}{
				"sys.ID": float64(1),
				name:     val,
			}
			require.NoError(FillElementFromJSON(data, cfg.Schemas.Schema(test.testCDoc), builder, cfg.app.Schemas()))
			o, err := builder.Build()
			require.ErrorIs(err, ErrWrongFieldType)
			require.Nil(o)
		}
	})

	t.Run("container errors", func(t *testing.T) {
		builder := NewIObjectBuilder(cfgs[istructs.AppQName_test1_app1], test.testCDoc)
		cases := []struct {
			f string
			v interface{}
		}{
			{"unknownContainer", []interface{}{}},
			{"record", []interface{}{"str"}},
			{"record", []interface{}{map[string]interface{}{"unknwonContainer": []interface{}{}}}},
		}
		cfg := cfgs[test.appName]
		for _, c := range cases {
			data := map[string]interface{}{
				c.f: c.v,
			}
			err := FillElementFromJSON(data, cfg.Schemas.Schema(test.testCDoc), builder, cfg.app.Schemas())
			require.Error(err)
		}
	})
}

func TestIBucketsFromIAppStructs(t *testing.T) {
	require := require.New(t)

	cfgs := AppConfigsType{}
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
	funcQName := istructs.NewQName("my", "func")
	rlExpected := istructs.RateLimit{
		Period:                1,
		MaxAllowedPerDuration: 2,
	}
	cfg.FunctionRateLimits.AddAppLimit(funcQName, rlExpected)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	as, err := asp.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	buckets := IBucketsFromIAppStructs(as)
	bsActual, err := buckets.GetDefaultBucketsState(GetFunctionRateLimitName(funcQName, istructs.RateLimitKind_byApp))
	require.NoError(err)
	require.Equal(rlExpected.Period, bsActual.Period)
	require.Equal(irates.NumTokensType(rlExpected.MaxAllowedPerDuration), bsActual.MaxTokensPerPeriod)
}
