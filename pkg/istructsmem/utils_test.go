/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
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

func Test_prefixBytes(t *testing.T) {
	type args struct {
		bytes  []byte
		prefix []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "add system QNameID to PK",
			args: args{
				bytes:  []byte{0x01, 0x02},
				prefix: []interface{}{QNameIDForError},
			},
			want: []byte{0x00, 0x01, 0x01, 0x02},
		},
		{
			name: "add system view QNameID to PK",
			args: args{
				bytes:  []byte{0x01, 0x02},
				prefix: []interface{}{QNameIDSysSingletonIDs}, //0x0016 (22 decimals)
			},
			want: []byte{0x00, byte(QNameIDSysSingletonIDs), 0x01, 0x02},
		},
		{
			name: "add QNameID and WSID to PK",
			args: args{
				bytes:  []byte{0x01, 0x02},
				prefix: []interface{}{QNameID(0x0107), istructs.WSID(0xA7010203)},
			},
			want: []byte{0x01, 0x07, 0x00, 0x00, 0x00, 0x00, 0xA7, 0x01, 0x02, 0x03, 0x01, 0x02},
		},
		{
			name: "add QNameID and WSID to nil PK",
			args: args{
				bytes:  nil,
				prefix: []interface{}{QNameID(0x0107), istructs.WSID(0xA7010203)},
			},
			want: []byte{0x01, 0x07, 0x00, 0x00, 0x00, 0x00, 0xA7, 0x01, 0x02, 0x03},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prefixBytes(tt.args.bytes, tt.args.prefix...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prefixBytes() = %v, want %v", got, tt.want)
			}
		})
	}

	require.New(t).Panics(func() {
		bytes := []byte{0x01, 0x02}
		const value = 55 // unknown type size!
		_ = prefixBytes(bytes, value)
	}, "must panic if expand bytes slice by unknown/variable size values")
}

func Test_fullBytes(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil case",
			args: args{b: nil},
			want: true,
		},
		{
			name: "null len case",
			args: args{b: []byte{}},
			want: true,
		},
		{
			name: "full byte test",
			args: args{b: []byte{0xFF}},
			want: true,
		},
		{
			name: "full word test",
			args: args{b: []byte{0xFF, 0xFF}},
			want: true,
		},
		{
			name: "full long bytes test",
			args: args{b: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
			want: true,
		},
		{
			name: "negative test",
			args: args{b: []byte("bytes")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fullBytes(tt.args.b); got != tt.want {
				t.Errorf("fullBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rightMarginCCols(t *testing.T) {
	type args struct {
		cc []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "nil test",
			args: args{cc: nil},
			want: nil,
		},
		{
			name: "null len test",
			args: args{cc: []byte{}},
			want: nil,
		},
		{
			name: "full byte test",
			args: args{cc: []byte{0xFF}},
			want: nil,
		},
		{
			name: "full word test",
			args: args{cc: []byte{0xFF, 0xFF}},
			want: nil,
		},
		{
			name: "vulgaris test",
			args: args{cc: []byte{0x01, 0x02}},
			want: []byte{0x01, 0x03},
		},
		{
			name: "full-end test",
			args: args{cc: []byte{0x01, 0xFF}},
			want: []byte{0x02, 0x00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFinishCCols := rightMarginCCols(tt.args.cc); !reflect.DeepEqual(gotFinishCCols, tt.want) {
				t.Errorf("rangeCCols() = %v, want %v", gotFinishCCols, tt.want)
			}
		})
	}
}

func Test_validIdent(t *testing.T) {
	type args struct {
		ident string
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr error
	}{
		// negative tests
		{
			name:    "error if empty ident",
			args:    args{ident: ""},
			wantOk:  false,
			wantErr: ErrNameMissed,
		},
		{
			name:    "error if wrong first char",
			args:    args{ident: "🐧26"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong digit starts",
			args:    args{ident: "2abc"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong last char",
			args:    args{ident: "lookAt🐧"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong char anywhere",
			args:    args{ident: "This🐧isMy"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if starts from digit",
			args:    args{ident: "7zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces at begin",
			args:    args{ident: " zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces at end",
			args:    args{ident: "zip "},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces anywhere",
			args:    args{ident: "zip zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name: "error if too long",
			args: args{ident: func() string {
				sworm := "_"
				for i := 0; i < MaxIdentLen; i++ {
					sworm += "_"
				}
				return sworm
			}()},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		// positive tests
		{
			name:   "one letter must pass",
			args:   args{ident: "i"},
			wantOk: true,
		},
		{
			name:   "single underscore must pass",
			args:   args{ident: "_"},
			wantOk: true,
		},
		{
			name:   "starts from underscore must pass",
			args:   args{ident: "_test"},
			wantOk: true,
		},
		{
			name:   "vulgaris camel notation must pass",
			args:   args{ident: "thisIsIdent1"},
			wantOk: true,
		},
		{
			name:   "vulgaris snake notation must pass",
			args:   args{ident: "this_is_ident_2"},
			wantOk: true,
		},
		{
			name:   "mixed notation must pass",
			args:   args{ident: "useMix_4_fun"},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := validIdent(tt.args.ident)
			if gotOk != tt.wantOk {
				t.Errorf("validIdent() = %v, want %v", gotOk, tt.wantOk)
				return
			}
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("validIdent() error = %v, wantErr is nil", err)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("validIdent() error = %v not is %v", err, tt.wantErr)
					return
				}
			} else if tt.wantErr != nil {
				t.Errorf("validIdent() error = nil, wantErr - %v", tt.wantErr)
				return
			}
		})
	}
}

func Test_validQName(t *testing.T) {
	type args struct {
		qName istructs.QName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr bool
	}{
		{
			name:    "NullQName must pass",
			args:    args{qName: istructs.NullQName},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if missed package",
			args:    args{qName: istructs.NewQName("", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid package",
			args:    args{qName: istructs.NewQName("5", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if missed entity",
			args:    args{qName: istructs.NewQName("test", "")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid entity",
			args:    args{qName: istructs.NewQName("naked", "🔫")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if system QNames",
			args:    args{qName: istructs.QNameForError},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if vulgaris QName",
			args:    args{qName: istructs.NewQName("test", "test")},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := validQName(tt.args.qName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validQName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("validQName() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_sysField(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.QName",
			args: args{istructs.SystemField_QName},
			want: true,
		},
		{
			name: "true if sys.ID",
			args: args{istructs.SystemField_ID},
			want: true,
		},
		{
			name: "true if sys.ParentID",
			args: args{istructs.SystemField_ParentID},
			want: true,
		},
		{
			name: "true if sys.Container",
			args: args{istructs.SystemField_Container},
			want: true,
		},
		{
			name: "true if sys.IsActive",
			args: args{istructs.SystemField_IsActive},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if vulgaris user",
			args: args{"userField"},
			want: false,
		},
		{
			name: "false if curious user",
			args: args{"sys.user"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sysField(tt.args.name); got != tt.want {
				t.Errorf("sysField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sysContainer(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.pkey",
			args: args{istructs.SystemContainer_ViewPartitionKey},
			want: true,
		},
		{
			name: "true if sys.ccols",
			args: args{istructs.SystemContainer_ViewClusteringCols},
			want: true,
		},
		{
			name: "true if sys.val",
			args: args{istructs.SystemContainer_ViewValue},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if vulgaris user",
			args: args{"userContainer"},
			want: false,
		},
		{
			name: "false if curious user",
			args: args{"sys.user"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sysContainer(tt.args.name); got != tt.want {
				t.Errorf("sysContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestElementFillAndGet(t *testing.T) {
	require := require.New(t)
	cfgs := testAppConfigs()
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
			require.ErrorIs(err, coreutils.ErrFieldTypeMismatch)
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
	cfg := cfgs.AddConfig(test.appName)
	funcQName := istructs.NewQName("my", "func")
	rlExpected := istructs.RateLimit{
		Period:                1,
		MaxAllowedPerDuration: 2,
	}
	cfg.FunctionRateLimits.AddAppLimit(funcQName, rlExpected)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	as, err := asp.AppStructs(test.appName)
	require.NoError(err)
	buckets := IBucketsFromIAppStructs(as)
	bsActual, err := buckets.GetDefaultBucketsState(GetFunctionRateLimitName(funcQName, istructs.RateLimitKind_byApp))
	require.NoError(err)
	require.Equal(rlExpected.Period, bsActual.Period)
	require.Equal(irates.NumTokensType(rlExpected.MaxAllowedPerDuration), bsActual.MaxTokensPerPeriod)
}
