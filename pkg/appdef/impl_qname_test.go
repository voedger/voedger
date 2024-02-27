/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin (QName refactoring)
 */

package appdef

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"slices"
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

	require.NotNil(ParseQName("saleOrders"))
	log.Println(ParseQName("saleOrders"))
	require.NotNil(ParseQName("sale.orders."))

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
		require.Error(err)
		log.Println(err)
		require.Equal(NullQName, q)
	})

	t.Run("Two-bytes string", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"\""))
		require.Error(err)
		require.Equal(NullQName, q)

		log.Println(err)
	})

	t.Run("No dot", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"bcd\""))
		require.Error(err)
		require.Equal(NullQName, q)

		log.Println(err)
	})

	t.Run("Two dots", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"c..d\""))
		require.Error(err)

		log.Println(err)
	})

	t.Run("json unquoted", func(t *testing.T) {
		q := NewQName("a", "b")
		err = q.UnmarshalJSON([]byte("c.d"))
		require.Error(err)
		log.Println(err)
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
