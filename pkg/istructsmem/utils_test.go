/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/isequencer"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
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

func Test_recordKey(t *testing.T) {
	const ws = istructs.WSID(0xa1a2a3a4a5a6a7a8)
	pkPref := []byte{0, byte(consts.SysView_Records), 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}
	tests := []struct {
		name   string
		id     istructs.RecordID
		wantPk []byte
		wantCc []byte
	}{
		{
			name:   "null record must return {0} and {0}",
			id:     istructs.NullRecordID,
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 0},
			wantCc: []byte{0, 0},
		},
		{
			name:   "4095 must return {0} and {0x0F, 0xFF}",
			id:     istructs.RecordID(4095),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 0},
			wantCc: []byte{0x0F, 0xFF},
		},
		{
			name:   "4096 must return {1} and {0}",
			id:     istructs.RecordID(4096),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 1},
			wantCc: []byte{0, 0},
		},
		{
			name:   "4097 must return {1} and {1}",
			id:     istructs.RecordID(4097),
			wantPk: []byte{0, 0, 0, 0, 0, 0, 0, 1},
			wantCc: []byte{0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPk, gotCc := recordKey(ws, tt.id)
			wantPk := append(pkPref, tt.wantPk...)
			if !reflect.DeepEqual(gotPk, wantPk) {
				t.Errorf("splitRecordID() gotPk = %v, want %v", gotPk, tt.wantPk)
			}
			if !reflect.DeepEqual(gotCc, tt.wantCc) {
				t.Errorf("splitRecordID() gotCc = %v, want %v", gotCc, tt.wantCc)
			}
		})
	}
}

func TestObjectFillAndGet(t *testing.T) {
	require := require.New(t)
	test := newTest()

	as := test.AppStructs

	builder := as.ObjectBuilder(test.testCDoc)

	t.Run("basic", func(t *testing.T) {

		data := map[string]any{
			"sys.ID":  json.Number("7"),
			"int32":   json.Number("1"),
			"int64":   json.Number("2"),
			"float32": json.Number("3"),
			"float64": json.Number("4"),
			"bytes":   "BQY=", // []byte{5,6}
			"string":  "str",
			"QName":   "test.CDoc",
			"bool":    true,
			"record": []any{
				map[string]any{
					"sys.ID": json.Number("8"),
					"int32":  json.Number("6"),
				},
			},
		}
		builder.FillFromJSON(data)
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
		for c := range o.Children("record") {
			require.Equal(istructs.RecordID(8), c.AsRecordID("sys.ID"))
			require.Equal(int32(6), c.AsInt32("int32"))
			count++
		}
		require.Equal(1, count)
	})

	t.Run("type errors", func(t *testing.T) {
		cases := map[string]any{
			"int32":   "str",
			"int64":   "str",
			"float32": "str",
			"float64": "str",
			"bytes":   float64(2),
			"string":  float64(3),
			"QName":   float64(4),
			"bool":    "str",
			"record": []any{
				map[string]any{"int32": "str"},
			},
		}

		for name, val := range cases {
			builder := as.ObjectBuilder(test.testCDoc)
			data := map[string]any{
				appdef.SystemField_ID: json.Number("1"),
				name:                  val,
			}
			builder.FillFromJSON(data)
			o, err := builder.Build()
			require.Error(err, require.Is(ErrWrongFieldTypeError))
			require.Nil(o)
		}
	})

	t.Run("container errors", func(t *testing.T) {
		builder := as.ObjectBuilder(test.testCDoc)
		cases := []struct {
			f string
			v any
		}{
			{"unknownContainer", []any{}},
			{"record", []any{"str"}},
			{"record", []any{map[string]any{"unknownContainer": []any{}}}},
		}
		for _, c := range cases {
			data := map[string]any{
				c.f: c.v,
			}
			builder.FillFromJSON(data)
			_, err := builder.Build()
			require.Error(err)
		}
	})
}

func TestIBucketsFromIAppStructs(t *testing.T) {
	require := require.New(t)

	cfgs := AppConfigsType{}
	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	funcQName := appdef.NewQName("test", "myFunc")

	rlExpected := istructs.RateLimit{
		Period:                1,
		MaxAllowedPerDuration: 2,
	}
	cfg.FunctionRateLimits.AddAppLimit(funcQName, rlExpected)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)
	as, err := asp.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	buckets := IBucketsFromIAppStructs(as)
	bsActual, err := buckets.GetDefaultBucketsState(GetFunctionRateLimitName(funcQName, istructs.RateLimitKind_byApp))
	require.NoError(err)
	require.Equal(rlExpected.Period, bsActual.Period)
	require.EqualValues(rlExpected.MaxAllowedPerDuration, bsActual.MaxTokensPerPeriod)
}
