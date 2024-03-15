/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin (QName refactoring)
 */

package appdef

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage_QName(t *testing.T) {

	require := require.New(t)

	// Create from pkg + entity

	qname := NewQName("sale", "orders")
	require.Equal(NewQName("sale", "orders"), qname)
	require.Equal("sale", qname.Pkg())
	require.Equal("orders", qname.Entity())

	require.Equal("sale.orders", fmt.Sprint(qname))

	// Parse string

	qname2, err := ParseQName("sale.orders")
	require.NoError(err)
	require.Equal(qname, qname2)

	// Errors. Only one dot allowed

	{
		q, e := ParseQName("saleOrders")
		require.NotNil(q)
		require.Equal(NullQName, q)
		require.ErrorIs(e, ErrInvalidQNameStringRepresentation)
	}

	{
		q, e := ParseQName("saleOrders")
		require.NotNil(ParseQName("sale.orders."))
		require.NotNil(q)
		require.Equal(NullQName, q)
		require.ErrorIs(e, ErrInvalidQNameStringRepresentation)
	}
}

func TestBasicUsage_QName_JSon(t *testing.T) {

	require := require.New(t)

	t.Run("Marshall/Unmarshal QName", func(t *testing.T) {

		qname := NewQName("airs-bp", `Карлсон 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&qname)
		require.NoError(err)

		// Unmarshal

		var qname2 = QName{}
		err = json.Unmarshal(j, &qname2)
		require.NoError(err)

		// Compare
		require.Equal(qname, qname2)

		t.Run("UnmarshalText must do nothing", func(t *testing.T) {
			qname := NewQName("test", "name")
			require.NoError(qname.UnmarshalText([]byte(qname.String())))
		})
	})

	t.Run("Marshall/Unmarshal QName as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			QName       QName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			QName:       NewQName("p", `Карлсон 哇"呀呀`),
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
		expected := map[QName]bool{
			NewQName("sys", "my"):           true,
			NewQName("sys", `Карлсон 哇"呀呀`): true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[QName]bool{}
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestQName_Json_NullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshall/Unmarshal NullQName", func(t *testing.T) {

		qname := NullQName

		// Marshal

		j, err := json.Marshal(&qname)
		require.NoError(err)

		// Unmarshal

		var qname2 = QName{}
		err = json.Unmarshal(j, &qname2)
		require.NoError(err)

		// Compare
		require.Equal(qname, qname2)
	})
}

func TestQName_Compare(t *testing.T) {
	require := require.New(t)

	q1 := NewQName("pkg", "entity")
	q2 := NewQName("pkg", "entity")
	require.Equal(q1, q2)

	q3 := NewQName("pkg", "entity_1")
	require.NotEqual(q1, q3)

	q4 := NewQName("pkg_1", "entity")
	require.NotEqual(q2, q4)

	t.Run("test CompareQName()", func(t *testing.T) {
		require.Equal(0, CompareQName(q1, q2))
		require.Equal(-1, CompareQName(q1, q3))
		require.Equal(1, CompareQName(q3, q1))
		require.Equal(-1, CompareQName(q2, q4))
		require.Equal(1, CompareQName(q4, q2))
	})
}

func Test_NullQName(t *testing.T) {
	require := require.New(t)
	require.Equal(QName{}, NullQName)
}

func TestQName_UnmarshalInvalidString(t *testing.T) {
	require := require.New(t)

	var err error
	t.Run("Nill slice", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON(nil)
		require.ErrorIs(err, strconv.ErrSyntax)
		require.Equal(NullQName, q)
	})

	t.Run("Two-bytes string", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"\""))
		require.ErrorIs(err, ErrInvalidQNameStringRepresentation)
		require.Equal(NullQName, q)
	})

	t.Run("No dot", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"bcd\""))
		require.ErrorIs(err, ErrInvalidQNameStringRepresentation)
		require.ErrorContains(err, "bcd")
		require.Equal(NullQName, q)
	})

	t.Run("Two dots", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"c..d\""))
		require.ErrorIs(err, ErrInvalidQNameStringRepresentation)
		require.ErrorContains(err, "c..d")
		require.Equal(NullQName, q)
	})

	t.Run("json unquoted", func(t *testing.T) {
		q := NewQName("a", "b")
		err = q.UnmarshalJSON([]byte("c.d"))
		require.ErrorIs(err, strconv.ErrSyntax)
		require.Equal(NullQName, q)
	})
}

func TestValidQName(t *testing.T) {
	type args struct {
		qName QName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr bool
	}{
		{
			name:    "NullQName must pass",
			args:    args{qName: NullQName},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if missed package",
			args:    args{qName: NewQName("", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid package",
			args:    args{qName: NewQName("5", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if missed entity",
			args:    args{qName: NewQName("test", "")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid entity",
			args:    args{qName: NewQName("naked", "🔫")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "ok if basic QName",
			args:    args{qName: NewQName("test", "test")},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := ValidQName(tt.args.qName)
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
		want QName
	}{
		{".", args{"."}, NullQName},
		{"sys.error", args{"sys.error"}, NewQName("sys", "error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustParseQName(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustParseQName() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("panic if invalid QName", func(t *testing.T) {
		require.Panics(t, func() {
			MustParseQName("🔫")
		})
	})
}

func TestQNamesFrom(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"empty", []string{}, `[]`},
		{"NullQName", []string{"."}, `[.]`},
		{"sys.error", []string{"sys.error"}, `[sys.error]`},
		{"deduplicate", []string{"a.a", "a.a"}, `[a.a]`},
		{"sort by package", []string{"c.c", "b.b", "a.a"}, `[a.a b.b c.c]`},
		{"sort by entity", []string{"a.b", "a.c", "a.x", "a.a"}, `[a.a a.b a.c a.x]`},
		{"sort and deduplicate", []string{"b.b", "z.z", "b.b", "a.a", "z.b"}, `[a.a b.b z.b z.z]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := make([]QName, len(tt.args))
			for i, arg := range tt.args {
				names[i] = MustParseQName(arg)
			}

			q := QNamesFrom(names...)
			if got := fmt.Sprint(q); got != tt.want {
				t.Errorf("QNamesFrom(%v) = %v, want %v", tt.args, got, tt.want)
			}

			if !slices.IsSortedFunc(q, CompareQName) {
				t.Errorf("QNamesFrom(%v) is not sorted", tt.args)
			}

			for _, n := range names {
				if !q.Contains(n) {
					t.Errorf("QNamesFrom(%v).Contains(%v) returns false", tt.args, n)
				}
				i, ok := q.Find(n)
				if !ok {
					t.Errorf("QNamesFrom(%v).Find(%v) returns false", tt.args, n)
				}
				if q[i] != n {
					t.Errorf("QNamesFrom(%v).Find(%v) returns wrong index %v", tt.args, n, i)
				}
				if q.Contains(MustParseQName("test.unknown")) {
					t.Errorf("QNamesFrom(%v).Contains(test.unknown) returns true", tt.args)
				}
			}
		})
	}
}

func TestValidQNames(t *testing.T) {
	type args struct {
		qName []QName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr error
	}{
		{"should be ok with empty names", args{[]QName{}}, true, nil},
		{"should be ok with null name", args{[]QName{NullQName}}, true, nil},
		{"should be ok with valid names", args{[]QName{NewQName("test", "name1"), NewQName("test", "name2")}}, true, nil},
		{"should be error with invalid name", args{[]QName{NewQName("test", "name"), NewQName("naked", "🔫")}}, false, ErrInvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, gotErr := ValidQNames(tt.args.qName...)
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

	fqn := NewFullQName(pkgPath, entity)
	require.Equal(NewFullQName(pkgPath, entity), fqn)
	require.Equal(pkgPath, fqn.PkgPath())
	require.Equal(entity, fqn.Entity())

	require.Equal(asString, fmt.Sprint(fqn))

	// Parse string

	fqn2, err := ParseFullQName(asString)
	require.NoError(err)
	require.Equal(pkgPath, fqn2.PkgPath())
	require.Equal(entity, fqn2.Entity())
}

func TestBasicUsage_FullQName_JSon(t *testing.T) {

	require := require.New(t)

	t.Run("Marshall/Unmarshal FullQName", func(t *testing.T) {

		fqn := NewFullQName("untill.pro/airs-bp", `Карлсон 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&fqn)
		require.NoError(err)

		// Unmarshal

		var fqn2 = FullQName{}
		err = json.Unmarshal(j, &fqn2)
		require.NoError(err)

		// Compare
		require.Equal(fqn, fqn2)

		t.Run("UnmarshalText must do nothing", func(t *testing.T) {
			qname := NewFullQName("test.test/test", "test")
			require.NoError(qname.UnmarshalText([]byte(qname.String())))
		})
	})

	t.Run("Marshall/Unmarshal as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			FullQName   FullQName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			FullQName:   NewFullQName("p.p/p", `Карлсон 哇"呀呀`),
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
		expected := map[FullQName]string{
			NewFullQName("test.test/test", "my"):           "one",
			NewFullQName("test.test/test", `Карлсон 哇"呀呀`): "two",
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[FullQName]string{}
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestFullQName_Json_NullFullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshall/Unmarshal NullFullQName", func(t *testing.T) {

		fqn := NullFullQName

		// Marshal

		j, err := json.Marshal(&fqn)
		require.NoError(err)

		// Unmarshal

		var fqn2 = FullQName{}
		err = json.Unmarshal(j, &fqn2)
		require.NoError(err)

		// Compare
		require.Equal(fqn, fqn2)
	})
}

func TestFullQName_Compare(t *testing.T) {
	require := require.New(t)

	fqn1 := NewFullQName("test.test/pkg", "entity")
	fqn2 := NewFullQName("test.test/pkg", "entity")
	require.Equal(fqn1, fqn2)

	fqn3 := NewFullQName("test.test/pkg", "entity_1")
	require.NotEqual(fqn1, fqn3)

	fqn4 := NewFullQName("test.test/pkg_1", "entity")
	require.NotEqual(fqn2, fqn4)

	t.Run("test CompareFullQName()", func(t *testing.T) {
		require.Equal(0, CompareFullQName(fqn1, fqn2))
		require.Equal(-1, CompareFullQName(fqn1, fqn3))
		require.Equal(1, CompareFullQName(fqn3, fqn1))
		require.Equal(-1, CompareFullQName(fqn2, fqn4))
		require.Equal(1, CompareFullQName(fqn4, fqn2))
	})
}

func Test_NullFullQName(t *testing.T) {
	require := require.New(t)
	require.Equal(FullQName{}, NullFullQName)
}

func TestFullQName_UnmarshalInvalidString(t *testing.T) {
	require := require.New(t)

	var err error
	t.Run("Nill slice", func(t *testing.T) {
		fqn := NewFullQName("a.a/a", "b")

		err = fqn.UnmarshalJSON(nil)
		require.ErrorIs(err, strconv.ErrSyntax)
		require.Equal(NullFullQName, fqn)
	})

	t.Run("Two-bytes string", func(t *testing.T) {
		fqn := NewFullQName("a.a/a", "b")

		err = fqn.UnmarshalJSON([]byte("\"\""))
		require.ErrorIs(err, ErrInvalidQNameStringRepresentation)
		require.Equal(NullFullQName, fqn)
	})

	t.Run("No dot", func(t *testing.T) {
		fqn := NewFullQName("a.a/a", "b")

		err = fqn.UnmarshalJSON([]byte("\"bcd\""))
		require.ErrorIs(err, ErrInvalidQNameStringRepresentation)
		require.ErrorContains(err, "bcd")
		require.Equal(NullFullQName, fqn)
	})

	t.Run("json unquoted", func(t *testing.T) {
		fqn := NewFullQName("a.a/a", "b")
		err = fqn.UnmarshalJSON([]byte("c.c/c.d"))
		require.ErrorIs(err, strconv.ErrSyntax)
		require.Equal(NullFullQName, fqn)
	})
}

func TestValidFullQName(t *testing.T) {
	type args struct {
		qfn FullQName
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr bool
	}{
		{
			name:    "NullFullQName must pass",
			args:    args{qfn: NullFullQName},
			wantOk:  true,
			wantErr: false,
		},
		{
			name:    "error if missed package",
			args:    args{qfn: NewFullQName("", "test")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if missed entity",
			args:    args{qfn: NewFullQName("test.test/test", "")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "error if invalid entity",
			args:    args{qfn: NewFullQName("test.test/naked", "🔫")},
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "ok if basic QName",
			args:    args{qfn: NewFullQName("test.test/test", "test")},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := ValidFullQName(tt.args.qfn)
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
		want FullQName
	}{
		{".", args{"."}, NullFullQName},
		{"test.test/test.error", args{"test.test/test.error"}, NewFullQName("test.test/test", "error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustParseFullQName(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustParseFullQName() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("panic if invalid FullQName", func(t *testing.T) {
		require.Panics(t, func() {
			MustParseFullQName("🔫")
		})
	})
}
