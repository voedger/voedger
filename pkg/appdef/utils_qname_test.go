/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin (QName refactoring)
 */

package appdef_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_ValidIdent(t *testing.T) {
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
			wantErr: appdef.ErrMissedError,
		},
		{
			name:    "error if wrong first char",
			args:    args{ident: "🐧26"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if wrong digit starts",
			args:    args{ident: "2abc"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if wrong last char",
			args:    args{ident: "lookAt🐧"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if wrong char anywhere",
			args:    args{ident: "This🐧isMy"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if starts from digit",
			args:    args{ident: "7zip"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if spaces at begin",
			args:    args{ident: " zip"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if spaces at end",
			args:    args{ident: "zip "},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if spaces anywhere",
			args:    args{ident: "zip zip"},
			wantOk:  false,
			wantErr: appdef.ErrInvalidError,
		},
		{
			name:    "error if too long",
			args:    args{ident: strings.Repeat("_", appdef.MaxIdentLen) + `_`},
			wantOk:  false,
			wantErr: appdef.ErrOutOfBoundsError,
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
			name:   "buck at any pos must pass",
			args:   args{ident: "test$test"},
			wantOk: true,
		},
		{
			name:   "buck at first pos must pass",
			args:   args{ident: "$test"},
			wantOk: true,
		},
		{
			name:   "basic camel notation must pass",
			args:   args{ident: "thisIsIdent1"},
			wantOk: true,
		},
		{
			name:   "basic snake notation must pass",
			args:   args{ident: "this_is_ident_2"},
			wantOk: true,
		},
		{
			name:   "mixed notation must pass",
			args:   args{ident: "useMix_4_fun$sense"},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := appdef.ValidIdent(tt.args.ident)
			if gotOk != tt.wantOk {
				t.Errorf("ValidIdent() = %v, want %v", gotOk, tt.wantOk)
				return
			}
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("ValidIdent() error = %v, wantErr is nil", err)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidIdent() error = %v not is %v", err, tt.wantErr)
					return
				}
			} else if tt.wantErr != nil {
				t.Errorf("ValidIdent() error = nil, wantErr - %v", tt.wantErr)
				return
			}
		})
	}
}

func TestBasicUsage_QName(t *testing.T) {

	require := require.New(t)

	// Create from pkg + entity

	qname := appdef.NewQName("sale", "orders")
	require.Equal(appdef.NewQName("sale", "orders"), qname)
	require.Equal("sale", qname.Pkg())
	require.Equal("orders", qname.Entity())

	require.Equal("sale.orders", fmt.Sprint(qname))

	// Parse string

	qname2, err := appdef.ParseQName("sale.orders")
	require.NoError(err)
	require.Equal(qname, qname2)

	// Errors. Only one dot allowed

	{
		q, e := appdef.ParseQName("saleOrders")
		require.NotNil(q)
		require.Equal(appdef.NullQName, q)
		require.ErrorIs(e, appdef.ErrConvertError)
	}

	{
		q, e := appdef.ParseQName("saleOrders")
		require.NotNil(appdef.ParseQName("sale.orders."))
		require.NotNil(q)
		require.Equal(appdef.NullQName, q)
		require.ErrorIs(e, appdef.ErrConvertError)
	}
}

func TestBasicUsage_QName_JSon(t *testing.T) {

	require := require.New(t)

	t.Run("Marshal/Unmarshal QName", func(t *testing.T) {

		qname := appdef.NewQName("airs-bp", `Carlson 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&qname)
		require.NoError(err)

		// Unmarshal

		var qname2 = appdef.QName{}
		err = json.Unmarshal(j, &qname2)
		require.NoError(err)

		// Compare
		require.Equal(qname, qname2)

		t.Run("UnmarshalText must do nothing", func(t *testing.T) {
			qname := appdef.NewQName("test", "name")
			require.NoError(qname.UnmarshalText([]byte(qname.String())))
		})
	})

	t.Run("Marshal/Unmarshal QName as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			QName       appdef.QName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			QName:       appdef.NewQName("p", `Carlson 哇"呀呀`),
			StringValue: "sv",
			IntValue:    56,
		}

		// Marshal

		j, err := json.Marshal(&ms)
		require.NoError(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.NoError(err)

		// Compare
		require.Equal(ms, ms2)
	})

	t.Run("key of a map", func(t *testing.T) {
		expected := map[appdef.QName]bool{
			appdef.NewQName("sys", "my"):           true,
			appdef.NewQName("sys", `Carlson 哇"呀呀`): true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[appdef.QName]bool{}
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestQName_Json_NullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshal/Unmarshal appdef.NullQName", func(t *testing.T) {

		qname := appdef.NullQName

		// Marshal

		j, err := json.Marshal(&qname)
		require.NoError(err)

		// Unmarshal

		var qname2 = appdef.QName{}
		err = json.Unmarshal(j, &qname2)
		require.NoError(err)

		// Compare
		require.Equal(qname, qname2)
	})
}

func TestQName_Compare(t *testing.T) {
	require := require.New(t)

	q1 := appdef.NewQName("pkg", "entity")
	q2 := appdef.NewQName("pkg", "entity")
	require.Equal(q1, q2)

	q3 := appdef.NewQName("pkg", "entity_1")
	require.NotEqual(q1, q3)

	q4 := appdef.NewQName("pkg_1", "entity")
	require.NotEqual(q2, q4)

	t.Run("test CompareQName()", func(t *testing.T) {
		require.Equal(0, appdef.CompareQName(q1, q2))
		require.Equal(-1, appdef.CompareQName(q1, q3))
		require.Equal(1, appdef.CompareQName(q3, q1))
		require.Equal(-1, appdef.CompareQName(q2, q4))
		require.Equal(1, appdef.CompareQName(q4, q2))
	})
}

func Test_NullQName(t *testing.T) {
	require := require.New(t)
	require.Equal(appdef.QName{}, appdef.NullQName)
}

func TestQName_UnmarshalInvalidString(t *testing.T) {
	tests := []struct {
		name        string
		value       []byte
		err         error
		errContains string
	}{
		{"Nill slice", nil, strconv.ErrSyntax, ""},
		{"Two quotes string", []byte(`""`), appdef.ErrConvertError, ""},
		{"No qualifier char `.`", []byte(`"bcd"`), appdef.ErrConvertError, "bcd"},
		{"Two `.`", []byte(`"c..d"`), appdef.ErrConvertError, "c..d"},
		{"json unquoted", []byte(`c.d`), strconv.ErrSyntax, ""},
	}

	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := appdef.NewQName("a", "b")
			err := n.UnmarshalJSON(tt.value)
			require.Equal(appdef.NullQName, n)
			require.ErrorIs(err, tt.err)
			if tt.errContains != "" {
				require.ErrorContains(err, tt.errContains)
			}
		})
	}
}

func TestValidQName(t *testing.T) {
	type args struct {
		qName appdef.QName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr bool
	}{
		{
			name:    "appdef.NullQName must pass",
			args:    args{qName: appdef.NullQName},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if missed package",
			args:    args{qName: appdef.NewQName("", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid package",
			args:    args{qName: appdef.NewQName("5", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if missed entity",
			args:    args{qName: appdef.NewQName("test", "")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid entity",
			args:    args{qName: appdef.NewQName("naked", "🔫")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "ok if basic QName",
			args:    args{qName: appdef.NewQName("test", "test")},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := appdef.ValidQName(tt.args.qName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidQName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("ValidQName() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestMustParseQName(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want appdef.QName
	}{
		{".", args{"."}, appdef.NullQName},
		{"sys.error", args{"sys.error"}, appdef.NewQName("sys", "error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appdef.MustParseQName(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustParseQName() = %v, want %v", got, tt.want)
			}
		})
	}

	require := require.New(t)
	t.Run("panic if invalid QName", func(t *testing.T) {
		require.Panics(func() { appdef.MustParseQName("🔫") }, require.Is(appdef.ErrConvertError), require.Has("🔫"))
	})
}

func TestQNames(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"empty", []string{}, `[]`},
		{"appdef.NullQName", []string{"."}, `[.]`},
		{"sys.error", []string{"sys.error"}, `[sys.error]`},
		{"deduplicate", []string{"a.a", "a.a"}, `[a.a]`},
		{"sort by package", []string{"c.c", "b.b", "a.a"}, `[a.a b.b c.c]`},
		{"sort by entity", []string{"a.b", "a.c", "a.x", "a.a"}, `[a.a a.b a.c a.x]`},
		{"sort and deduplicate", []string{"b.b", "z.z", "b.b", "a.a", "z.b"}, `[a.a b.b z.b z.z]`},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := make([]appdef.QName, len(tt.args))
			for i, arg := range tt.args {
				names[i] = appdef.MustParseQName(arg)
			}

			q := appdef.QNamesFrom(names...)
			require.Equal(tt.want, fmt.Sprint(q), "appdef.QNamesFrom(%v) = %v, want %v", tt.args, fmt.Sprint(q), tt.want)

			require.True(slices.IsSortedFunc(q, appdef.CompareQName), "appdef.QNamesFrom(%v) is not sorted", tt.args)

			for _, n := range names {
				i, ok := q.Find(n)
				require.True(ok, "appdef.QNamesFrom(%v).Find(%v) returns false", tt.args, n)
				require.Equal(n, q[i], "appdef.QNamesFrom(%v).Find(%v) returns wrong index %v", tt.args, n, i)

				require.True(q.Contains(n), "appdef.QNamesFrom(%v).Contains(%v) returns false", tt.args, n)

				unk := appdef.MustParseQName("test.unknown")
				require.False(q.Contains(unk), "appdef.QNamesFrom(%v).Contains(test.unknown) returns true", tt.args)

				require.False(q.ContainsAll(n, unk), "appdef.QNamesFrom(%v).ContainsAll(%v, %v) returns true", tt.args, n, unk)
				require.True(q.ContainsAll(names[0], n), "appdef.QNamesFrom(%v).ContainsAll(%v, %v) returns false", tt.args, names[0], n)

				require.False(q.ContainsAny(unk), "appdef.QNamesFrom(%v).ContainsAny(%v) returns true", tt.args, unk)
				require.True(q.ContainsAny(n, unk), "appdef.QNamesFrom(%v).ContainsAny(%v, %v) returns false", tt.args, n, unk)
			}

			t.Run("test Collect", func(t *testing.T) {
				c := appdef.CollectQNames(slices.Values(names))
				require.Equal(q, c)
			})
		})
	}
}

func TestValidQNames(t *testing.T) {
	type args struct {
		qName []appdef.QName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr error
	}{
		{"should be ok with empty names", args{[]appdef.QName{}}, true, nil},
		{"should be ok with null name", args{[]appdef.QName{appdef.NullQName}}, true, nil},
		{"should be ok with valid names", args{[]appdef.QName{appdef.NewQName("test", "name1"), appdef.NewQName("test", "name2")}}, true, nil},
		{"should be error with invalid name", args{[]appdef.QName{appdef.NewQName("test", "name"), appdef.NewQName("naked", "🔫")}}, false, appdef.ErrInvalidError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, gotErr := appdef.ValidQNames(tt.args.qName...)
			if gotOk != tt.wantOk {
				t.Errorf("ValidQNames() = %v, want %v", gotOk, tt.wantOk)
			}
			if tt.wantErr == nil {
				if gotErr != nil {
					t.Errorf("unexpected error %v for ValidQNames(%v)", gotErr, tt.args.qName)
				}
			}
			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("expected error %v, but no error for ValidQNames(%v)", tt.wantErr, tt.args.qName)
				} else if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("ValidQNames(%v) returns error «%v» which is not expected error «%v»", tt.args.qName, gotErr, tt.wantErr)
				}
			}
		})
	}
}

func TestBasicUsage_FullQName(t *testing.T) {

	require := require.New(t)

	// Create from pkgPath + entity

	const (
		pkgPath = "github.com/test/sale"
		entity  = "orders"

		asString = "github.com/test/sale.orders"
	)

	fqn := appdef.NewFullQName(pkgPath, entity)
	require.Equal(appdef.NewFullQName(pkgPath, entity), fqn)
	require.Equal(pkgPath, fqn.PkgPath())
	require.Equal(entity, fqn.Entity())

	require.Equal(asString, fmt.Sprint(fqn))

	// Parse string

	fqn2, err := appdef.ParseFullQName(asString)
	require.NoError(err)
	require.Equal(pkgPath, fqn2.PkgPath())
	require.Equal(entity, fqn2.Entity())
}

func TestBasicUsage_FullQName_JSon(t *testing.T) {

	require := require.New(t)

	t.Run("Marshal/Unmarshal FullQName", func(t *testing.T) {

		fqn := appdef.NewFullQName("untill.pro/airs-bp", `Carlson 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&fqn)
		require.NoError(err)

		// Unmarshal

		var fqn2 = appdef.FullQName{}
		err = json.Unmarshal(j, &fqn2)
		require.NoError(err)

		// Compare
		require.Equal(fqn, fqn2)

		t.Run("UnmarshalText must do nothing", func(t *testing.T) {
			qname := appdef.NewFullQName("test.test/test", "test")
			require.NoError(qname.UnmarshalText([]byte(qname.String())))
		})
	})

	t.Run("Marshal/Unmarshal as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			FullQName   appdef.FullQName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			FullQName:   appdef.NewFullQName("p.p/p", `Carlson 哇"呀呀`),
			StringValue: "sv",
			IntValue:    56,
		}

		// Marshal

		j, err := json.Marshal(&ms)
		require.NoError(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.NoError(err)

		// Compare
		require.Equal(ms, ms2)
	})

	t.Run("key of a map", func(t *testing.T) {
		expected := map[appdef.FullQName]string{
			appdef.NewFullQName("test.test/test", "my"):           "one",
			appdef.NewFullQName("test.test/test", `Carlson 哇"呀呀`): "two",
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[appdef.FullQName]string{}
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestFullQName_Json_NullFullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshal/Unmarshal NullFullQName", func(t *testing.T) {

		fqn := appdef.NullFullQName

		// Marshal

		j, err := json.Marshal(&fqn)
		require.NoError(err)

		// Unmarshal

		var fqn2 = appdef.FullQName{}
		err = json.Unmarshal(j, &fqn2)
		require.NoError(err)

		// Compare
		require.Equal(fqn, fqn2)
	})
}

func TestFullQName_Compare(t *testing.T) {
	require := require.New(t)

	fqn1 := appdef.NewFullQName("test.test/pkg", "entity")
	fqn2 := appdef.NewFullQName("test.test/pkg", "entity")
	require.Equal(fqn1, fqn2)

	fqn3 := appdef.NewFullQName("test.test/pkg", "entity_1")
	require.NotEqual(fqn1, fqn3)

	fqn4 := appdef.NewFullQName("test.test/pkg_1", "entity")
	require.NotEqual(fqn2, fqn4)

	t.Run("test CompareFullQName()", func(t *testing.T) {
		require.Equal(0, appdef.CompareFullQName(fqn1, fqn2))
		require.Equal(-1, appdef.CompareFullQName(fqn1, fqn3))
		require.Equal(1, appdef.CompareFullQName(fqn3, fqn1))
		require.Equal(-1, appdef.CompareFullQName(fqn2, fqn4))
		require.Equal(1, appdef.CompareFullQName(fqn4, fqn2))
	})
}

func Test_NullFullQName(t *testing.T) {
	require := require.New(t)
	require.Equal(appdef.FullQName{}, appdef.NullFullQName)
}

func TestFullQName_UnmarshalInvalidString(t *testing.T) {
	tests := []struct {
		name        string
		value       []byte
		err         error
		errContains string
	}{
		{"Nill slice", nil, strconv.ErrSyntax, ""},
		{"Two quotes string", []byte(`""`), appdef.ErrConvertError, ""},
		{"No qualifier char `.`", []byte(`"test/test"`), appdef.ErrConvertError, "test/test"},
		{"json unquoted", []byte(`test/test.name`), strconv.ErrSyntax, ""},
	}

	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := appdef.NewFullQName("a", "b")
			err := n.UnmarshalJSON(tt.value)
			require.Equal(appdef.NullFullQName, n)
			require.ErrorIs(err, tt.err)
			if tt.errContains != "" {
				require.ErrorContains(err, tt.errContains)
			}
		})
	}
}

func TestValidFullQName(t *testing.T) {
	type args struct {
		qfn appdef.FullQName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr bool
	}{
		{
			name:    "appdef.NullFullQName must pass",
			args:    args{qfn: appdef.NullFullQName},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if missed package",
			args:    args{qfn: appdef.NewFullQName("", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if missed entity",
			args:    args{qfn: appdef.NewFullQName("test.test/test", "")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid entity",
			args:    args{qfn: appdef.NewFullQName("test.test/naked", "🔫")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "ok if basic QName",
			args:    args{qfn: appdef.NewFullQName("test.test/test", "test")},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := appdef.ValidFullQName(tt.args.qfn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidFullQName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("ValidFullQName() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestMustParseFullQName(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want appdef.FullQName
	}{
		{".", args{"."}, appdef.NullFullQName},
		{"test.test/test.error", args{"test.test/test.error"}, appdef.NewFullQName("test.test/test", "error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appdef.MustParseFullQName(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustParseFullQName() = %v, want %v", got, tt.want)
			}
		})
	}

	require := require.New(t)
	t.Run("panic if invalid FullQName", func(t *testing.T) {
		require.Panics(func() { appdef.MustParseFullQName("🔫") }, require.Is(appdef.ErrConvertError), require.Has("🔫"))
	})
}

func TestBasicUsage_AppQName(t *testing.T) {

	require := require.New(t)

	// Create from owner + name

	aqn := appdef.NewAppQName("sys", "registry")
	require.Equal(appdef.NewAppQName("sys", "registry"), aqn)
	require.Equal("sys", aqn.Owner())
	require.Equal("registry", aqn.Name())
	require.True(aqn.IsSys())

	require.Equal("sys/registry", fmt.Sprint(aqn))

	// Parse string

	aqn2, err := appdef.ParseAppQName("sys/registry")
	require.NoError(err)
	require.Equal(aqn, aqn2)

	// Errors. Only one slash allowed
	n, err := appdef.ParseAppQName("sys")
	require.Equal(appdef.NullAppQName, n)
	require.ErrorIs(err, appdef.ErrConvertError)

	n, err = appdef.ParseAppQName("sys/registry/")
	require.Equal(appdef.NullAppQName, n)
	require.ErrorIs(err, appdef.ErrConvertError)
}

func TestBasicUsage_AppQName_JSon(t *testing.T) {
	require := require.New(t)

	t.Run("Marshal/Unmarshal QName", func(t *testing.T) {

		aqn := appdef.NewAppQName("sys", `Carlson 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&aqn)
		require.NoError(err)

		// Unmarshal

		var aqn2 = appdef.AppQName{}
		err = json.Unmarshal(j, &aqn2)
		require.NoError(err)

		// Compare
		require.Equal(aqn, aqn2)

		t.Run("UnmarshalText must do nothing", func(t *testing.T) {
			aqn := appdef.NewAppQName("test", "name")
			require.NoError(aqn.UnmarshalText([]byte(aqn.String())))
		})
	})

	t.Run("Marshal/Unmarshal AppQName as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			AQN         appdef.AppQName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			AQN:         appdef.NewAppQName("p", `Carlson 哇"呀呀`),
			StringValue: "sv",
			IntValue:    56,
		}

		// Marshal

		j, err := json.Marshal(&ms)
		require.NoError(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.NoError(err)

		// Compare
		require.Equal(ms, ms2)
	})

	t.Run("key of a map", func(t *testing.T) {
		expected := map[appdef.AppQName]bool{
			appdef.NewAppQName("sys", "my"):           true,
			appdef.NewAppQName("sys", `Carlson 哇"呀呀`): true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[appdef.AppQName]bool{}
		//log.Println(string(b))
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestAppQName_IsSys(t *testing.T) {
	tests := []struct {
		aqn  appdef.AppQName
		want bool
	}{
		{appdef.NullAppQName, false},
		{appdef.MustParseAppQName("sys/registry"), true},
		{appdef.MustParseAppQName("owner/my"), false},
	}
	for _, tt := range tests {
		t.Run(tt.aqn.String(), func(t *testing.T) {
			if got := tt.aqn.IsSys(); got != tt.want {
				t.Errorf("AppQName.IsSys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustParseAppQName(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want appdef.AppQName
	}{
		{"/", "/", appdef.NullAppQName},
		{"sys/router", "sys/router", appdef.NewAppQName("sys", "router")},
		{"own/app", "own/app", appdef.NewAppQName("own", "app")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appdef.MustParseAppQName(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustParseAppQName(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	require := require.New(t)
	t.Run("panic if invalid AppQName", func(t *testing.T) {
		require.Panics(func() { appdef.MustParseAppQName("🔫") }, require.Is(appdef.ErrConvertError), require.Has("🔫"))
	})
}

func TestAppQName_Compare(t *testing.T) {
	require := require.New(t)

	q1_1 := appdef.NewAppQName("sys", "registry")
	q1_2 := appdef.NewAppQName("sys", "registry")
	require.Equal(q1_1, q1_2)

	q2 := appdef.NewQName("sys", "registry2")
	require.NotEqual(q1_1, q2)
}

func TestAppQName_Json_NullAppQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshal/Unmarshal appdef.NullAppQName", func(t *testing.T) {

		aqn := appdef.NullAppQName

		// Marshal

		j, err := json.Marshal(&aqn)
		require.NoError(err)

		// Unmarshal

		aqn2 := appdef.NewAppQName("sys", "registry")
		err = json.Unmarshal(j, &aqn2)
		require.NoError(err)

		// Compare
		require.Equal(aqn, aqn2)
	})
}

func TestAppQName_UnmarshalInvalidString(t *testing.T) {
	tests := []struct {
		name        string
		value       []byte
		err         error
		errContains string
	}{
		{"Nill slice", nil, strconv.ErrSyntax, ""},
		{"Two quotes string", []byte(`""`), appdef.ErrConvertError, ""},
		{"No qualifier char `/`", []byte(`"bcd"`), appdef.ErrConvertError, "bcd"},
		{"Two `/`", []byte(`"c//d"`), appdef.ErrConvertError, "c//d"},
		{"json unquoted", []byte(`c/d`), strconv.ErrSyntax, ""},
	}

	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := appdef.NewAppQName("a", "b")
			err := n.UnmarshalJSON(tt.value)
			require.Equal(appdef.NullAppQName, n)
			require.ErrorIs(err, tt.err)
			if tt.errContains != "" {
				require.ErrorContains(err, tt.errContains)
			}
		})
	}
}

func TestParseQNames(t *testing.T) {
	type args struct {
		val []string
	}
	tests := []struct {
		name    string
		args    args
		wantRes appdef.QNames
		wantErr bool
	}{
		{"empty", args{[]string{}}, appdef.QNames{}, false},
		{"appdef.NullQName", args{[]string{"."}}, appdef.QNames{appdef.NullQName}, false},
		{"sys.error", args{[]string{"sys.error"}}, appdef.QNames{appdef.NewQName("sys", "error")}, false},
		{"deduplicate", args{[]string{"a.a", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a")}, false},
		{"sort by package", args{[]string{"c.c", "b.b", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("b", "b"), appdef.NewQName("c", "c")}, false},
		{"sort by entity", args{[]string{"a.b", "a.c", "a.x", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("a", "b"), appdef.NewQName("a", "c"), appdef.NewQName("a", "x")}, false},
		{"sort and deduplicate", args{[]string{"b.b", "z.z", "b.b", "a.a", "z.b"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("b", "b"), appdef.NewQName("z", "b"), appdef.NewQName("z", "z")}, false},
		// Errors
		{"error if invalid qname", args{[]string{"naked 🔫"}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := appdef.ParseQNames(tt.args.val...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("ParseQNames() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestMustParseQNames(t *testing.T) {
	type args struct {
		val []string
	}
	tests := []struct {
		name       string
		args       args
		want       appdef.QNames
		wantPanics bool
	}{
		{"empty", args{[]string{}}, appdef.QNames{}, false},
		{"sys.error", args{[]string{"sys.error"}}, appdef.QNames{appdef.NewQName("sys", "error")}, false},
		{"deduplicate", args{[]string{"a.a", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a")}, false},
		{"sort by package", args{[]string{"c.c", "b.b", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("b", "b"), appdef.NewQName("c", "c")}, false},
		{"sort by entity", args{[]string{"a.b", "a.c", "a.x", "a.a"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("a", "b"), appdef.NewQName("a", "c"), appdef.NewQName("a", "x")}, false},
		{"sort and deduplicate", args{[]string{"b.b", "z.z", "b.b", "a.a", "z.b"}}, appdef.QNames{appdef.NewQName("a", "a"), appdef.NewQName("b", "b"), appdef.NewQName("z", "b"), appdef.NewQName("z", "z")}, false},
		// Errors
		{"panic if invalid qname", args{[]string{"naked 🔫"}}, nil, true},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanics {
				require.Panics(func() { appdef.MustParseQNames(tt.args.val...) })
			} else {
				if got := appdef.MustParseQNames(tt.args.val...); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("MustParseQNames() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
