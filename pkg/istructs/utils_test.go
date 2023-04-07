/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin (I*Function implementation tests)
 * @author: Maxim Geraskin (QName refactoring)
 * @author: Maxim Geraskin Null* objects implementation
 * @author: Maxim Geraskin AppQName
 */

package istructs

import (
	"encoding/json"
	"fmt"
	"log"
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
	require.Nil(err)
	require.Equal(qname, qname2)

	// Errors. Only one dot allowed

	require.NotNil(ParseQName("saleorders"))
	log.Println(ParseQName("saleorders"))
	require.NotNil(ParseQName("sale.orders."))

}

func TestBasicUsage_QName_JSon(t *testing.T) {

	require := require.New(t)

	t.Run("Marshall/unmarshall QName", func(t *testing.T) {

		qname := NewQName("airs-bp", `Карлосон 哇"呀呀`)

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

	t.Run("Marshall/unmarshall QName as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			QName       QName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			QName:       NewQName("p", `Карлосон 哇"呀呀`),
			StringValue: "sv",
			IntValue:    56,
		}

		// Marshal

		j, err := json.Marshal(&ms)
		require.Nil(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.Nil(err)

		// Compare
		require.Equal(ms, ms2)
	})

	t.Run("key of a map", func(t *testing.T) {
		expected := map[QName]bool{
			NewQName("sys", "my"):            true,
			NewQName("sys", `Карлосон 哇"呀呀`): true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[QName]bool{}
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestBasicUsage_AppQName(t *testing.T) {

	require := require.New(t)

	// Create from onwer + name

	appqname := NewAppQName("sys", "registry")
	require.Equal(NewAppQName("sys", "registry"), appqname)
	require.Equal("sys", appqname.Owner())
	require.Equal("registry", appqname.Name())

	require.Equal("sys/registry", fmt.Sprint(appqname))

	// Parse string

	appqname2, err := ParseAppQName("sys/registry")
	require.Nil(err)
	require.Equal(appqname, appqname2)

	// Errors. Only one slash allowed

	require.NotNil(ParseAppQName("sys"))
	log.Println(ParseAppQName("sys"))
	require.NotNil(ParseAppQName("sys/registry/"))
}

func TestBasicUsage_AppQName_JSon(t *testing.T) {
	require := require.New(t)

	t.Run("Marshall/unmarshall QName", func(t *testing.T) {

		appqname := NewAppQName("sys", `Карлосон 哇"呀呀`)

		// Marshal

		j, err := json.Marshal(&appqname)
		require.Nil(err)

		// Unmarshal

		var appqname2 = AppQName{}
		err = json.Unmarshal(j, &appqname2)
		require.Nil(err)

		// Compare
		require.Equal(appqname, appqname2)
	})

	t.Run("Marshall/unmarshall AppQName as a part of the structure", func(t *testing.T) {

		type myStruct struct {
			AQN         AppQName
			StringValue string
			IntValue    int
		}

		ms := myStruct{
			AQN:         NewAppQName("p", `Карлосон 哇"呀呀`),
			StringValue: "sv",
			IntValue:    56,
		}

		// Marshal

		j, err := json.Marshal(&ms)
		require.Nil(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.Nil(err)

		// Compare
		require.Equal(ms, ms2)
	})

	t.Run("key of a map", func(t *testing.T) {
		expected := map[AppQName]bool{
			NewAppQName("sys", "my"):            true,
			NewAppQName("sys", `Карлосон 哇"呀呀`): true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		actual := map[AppQName]bool{}
		log.Println(string(b))
		require.NoError(json.Unmarshal(b, &actual))
		require.Equal(expected, actual)
	})
}

func TestBasicUsage_RecordID(t *testing.T) {

	var id RecordID

	// assign temporary value to id
	id = 1
	require.True(t, id.IsRaw())

	if id.IsRaw() {
		// call permanent id generator
		id = regenerateID(id)

		require.False(t, id.IsRaw())
	}
}

func TestBasicUsage_NewWSID(t *testing.T) {
	require := require.New(t)

	// First workspace in first cluster
	{
		wsid := NewWSID(NullClusterID+1, NullWSID+1)
		require.Equal(WSID(0x0000_8000_0000_0001), wsid)
		require.Equal(NullClusterID+1, wsid.ClusterID())
		require.Equal(NullWSID+1, wsid.BaseWSID())
	}

	// Third workspace in second cluster
	{
		wsid := NewWSID(NullClusterID+2, NullWSID+3)
		require.Equal(WSID(0x0001_0000_0000_0003), wsid)
		require.Equal(NullClusterID+2, wsid.ClusterID())
		require.Equal(NullWSID+3, wsid.BaseWSID())
	}

	// Max possible workspace in max possible cluster
	{
		wsid := NewWSID(MaxClusterID, 0x7fffffffffff)
		require.Equal(WSID(0x7fff_ffff_ffff_ffff), wsid)
		require.Equal(ClusterID(0xffff), wsid.ClusterID())
		require.Equal(WSID(0x7fffffffffff), wsid.BaseWSID())
	}
}

func TestBasicUsage_NewRecordID(t *testing.T) {
	require := require.New(t)

	// First cluster-generated ID
	{
		recordID := NewRecordID(1)
		require.Equal(RecordID(ClusterAsRegisterID)*RegisterFactor+1, recordID)
	}
	// Second cluster-generated ID
	{
		recordID := NewRecordID(2)
		require.Equal(RecordID(ClusterAsRegisterID)*RegisterFactor+2, recordID)
	}
}

func TestBasicUsage_NewCDocCRecordID(t *testing.T) {
	require := require.New(t)

	// First cluster-generated C*- ID
	{
		recordID := NewCDocCRecordID(1)
		require.Equal(RecordID(ClusterAsCRecordRegisterID)*RegisterFactor+1, recordID)
	}
	// Second cluster-generated ID
	{
		recordID := NewCDocCRecordID(2)
		require.Equal(RecordID(ClusterAsCRecordRegisterID)*RegisterFactor+2, recordID)
	}
}

func TestBasicUsage_BaseRecordID(t *testing.T) {
	require := require.New(t)

	// First cluster-generated ID
	{
		recordID := NewRecordID(1)
		require.Equal(RecordID(ClusterAsRegisterID)*RegisterFactor+1, recordID)
		require.Equal(RecordID(1), recordID.BaseRecordID())
	}
	// Second cluster-generated ID
	{
		recordID := NewRecordID(2)
		require.Equal(RecordID(ClusterAsRegisterID)*RegisterFactor+2, recordID)
		require.Equal(RecordID(2), recordID.BaseRecordID())
	}
}

// regenerateID: just example for test usage
func regenerateID(id RecordID) RecordID {
	const increment = MaxRawRecordID + 1
	if id.IsRaw() {
		return id + increment
	}
	return id
}

func TestQName_Json_NullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshall/unmarshall NullQName", func(t *testing.T) {

		qname := NullQName

		// Marshal

		j, err := json.Marshal(&qname)
		require.Nil(err)

		// Unmarshal

		var qname2 = QName{}
		err = json.Unmarshal(j, &qname2)
		require.Nil(err)

		// Compare
		require.Equal(qname, qname2)
	})
}

func TestQName_Compare(t *testing.T) {
	require := require.New(t)

	q1 := NewQName(SysPackage, "Error")
	require.Equal(QNameForError, q1)
	require.True(QNameForError == q1)

	q2 := NewQName(SysPackage, "error2")
	require.NotEqual(q1, q2)
}

func TestAppQName_Compare(t *testing.T) {
	require := require.New(t)

	q1_1 := NewAppQName("sys", "registry")
	q1_2 := NewAppQName("sys", "registry")
	require.Equal(q1_1, q1_2)
	require.True(q1_1 == q1_2)

	q2 := NewQName("sys", "registry2")
	require.NotEqual(q1_1, q2)
}

func TestAppQName_Json_NullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshall/unmarshall NullQName", func(t *testing.T) {

		aqn := NullAppQName

		// Marshal

		j, err := json.Marshal(&aqn)
		require.Nil(err)

		// Unmarshal

		var aqn2 = AppQName{}
		err = json.Unmarshal(j, &aqn2)
		require.Nil(err)

		// Compare
		require.Equal(aqn, aqn2)
	})
}

func TestAppQName_UnmarshalInvalidString(t *testing.T) {
	require := require.New(t)

	var err error
	t.Run("Nill slice", func(t *testing.T) {
		q := NewAppQName("a", "b")

		err = q.UnmarshalJSON(nil)
		require.NotNil(err)
		log.Println(err)
		require.Equal(NullAppQName, q)
	})

	t.Run("Two-bytes string", func(t *testing.T) {
		q := NewAppQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"\""))
		require.NotNil(err)
		require.Equal(NullAppQName, q)

		log.Println(err)
	})

	t.Run("No dot", func(t *testing.T) {
		q := NewAppQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"bcd\""))
		require.NotNil(err)
		require.Equal(NullAppQName, q)

		log.Println(err)
	})

	t.Run("Two dots", func(t *testing.T) {
		q := NewAppQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"c..d\""))
		require.NotNil(err)
		log.Println(err)
	})

	t.Run("json unquoted", func(t *testing.T) {
		q := NewAppQName("a", "b")
		err = q.UnmarshalJSON([]byte("c.d"))
		require.NotNil(err)
		log.Println(err)
	})

}

func TestQName_UnmarshalInvalidString(t *testing.T) {
	require := require.New(t)

	var err error
	t.Run("Nill slice", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON(nil)
		require.NotNil(err)
		log.Println(err)
		require.Equal(NullQName, q)
	})

	t.Run("Two-bytes string", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"\""))
		require.NotNil(err)
		require.Equal(NullQName, q)

		log.Println(err)
	})

	t.Run("No dot", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"bcd\""))
		require.NotNil(err)
		require.Equal(NullQName, q)

		log.Println(err)
	})

	t.Run("Two dots", func(t *testing.T) {
		q := NewQName("a", "b")

		err = q.UnmarshalJSON([]byte("\"c..d\""))
		require.NotNil(err)

		log.Println(err)
	})

	t.Run("json unquoted", func(t *testing.T) {
		q := NewQName("a", "b")
		err = q.UnmarshalJSON([]byte("c.d"))
		require.NotNil(err)
		log.Println(err)
	})
}

func TestRecordID_IsTemp(t *testing.T) {
	tests := []struct {
		name string
		id   RecordID
		want bool
	}{
		{"test basic usage", 1, true},
		{"test basic usage perm id", 725246548, false},
		{"test zero value", 0, false},
		{"test max range value", 0xFFFF, true},
		{"test min out of range value", 0xFFFF + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsRaw(); got != tt.want {
				t.Errorf("RecordID(%v).IsTemp() = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestNullObject(t *testing.T) {
	require := require.New(t)
	null := NewNullObject()

	require.Nil(null.AsBytes(NullName))
	require.Equal(float32(0), null.AsFloat32(NullName))
	require.Equal(float64(0), null.AsFloat64(NullName))
	require.Equal(int32(0), null.AsInt32(NullName))
	require.Equal(int64(0), null.AsInt64(NullName))
	require.Equal("", null.AsString(NullName))

	require.Equal(NullQName, null.AsQName(NullName))
	require.Equal(false, null.AsBool(NullName))
	require.Equal(NullRecordID, null.AsRecordID(NullName))

	require.Equal(NullQName, null.QName())

	// Should not be called
	{
		null.Containers(nil)
		null.Elements(NullName, nil)
		null.RecordIDs(true, nil)
		null.FieldNames(nil)
	}

	t.Run("IRecord fields", func(t *testing.T) {
		r := null.AsRecord()
		require.Equal(NullQName, r.QName())
		require.Equal(NullQName, r.QName())
		require.Equal("", r.Container())
		require.Equal(NullRecordID, r.ID())
		require.Equal(NullRecordID, r.Parent())

	})

}

func TestContainerOccursType_String(t *testing.T) {
	tests := []struct {
		name string
		o    ContainerOccursType
		want string
	}{
		{
			name: "0 —> `0`",
			o:    0,
			want: `0`,
		},
		{
			name: "1 —> `1`",
			o:    1,
			want: `1`,
		},
		{
			name: "∞ —> `unbounded`",
			o:    ContainerOccurs_Unbounded,
			want: `unbounded`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.String(); got != tt.want {
				t.Errorf("ContainerOccursType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerOccursType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		o       ContainerOccursType
		want    string
		wantErr bool
	}{
		{
			name:    "0 —> `0`",
			o:       0,
			want:    `0`,
			wantErr: false,
		},
		{
			name:    "1 —> `1`",
			o:       1,
			want:    `1`,
			wantErr: false,
		},
		{
			name:    "∞ —> `unbounded`",
			o:       ContainerOccurs_Unbounded,
			want:    `"unbounded"`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContainerOccursType.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ContainerOccursType.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerOccursType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    ContainerOccursType
		wantErr bool
	}{
		{
			name:    "0 —> 0",
			data:    `0`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "1 —> 1",
			data:    `1`,
			want:    1,
			wantErr: false,
		},
		{
			name:    `"unbounded" —> ∞`,
			data:    `"unbounded"`,
			want:    ContainerOccurs_Unbounded,
			wantErr: false,
		},
		{
			name:    `"3" —> error`,
			data:    `"3"`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `65536 —> error`,
			data:    `65536`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `-1 —> error`,
			data:    `-1`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `"abc" —> error`,
			data:    `"abc"`,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o ContainerOccursType
			err := o.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("ContainerOccursType.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if o != tt.want {
					t.Errorf("o.UnmarshalJSON() result = %v, want %v", o, tt.want)
				}
			}
		})
	}
}

func TestDataKindType_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		i    DataKindType
		want string
	}{
		{
			name: `0 —> "DataKind_null"`,
			i:    DataKind_null,
			want: `DataKind_null`,
		},
		{
			name: `1 —> "DataKind_int32"`,
			i:    DataKind_int32,
			want: `DataKind_int32`,
		},
		{
			name: `DataKind_FakeLast —> 12`,
			i:    DataKind_FakeLast,
			want: strconv.FormatUint(uint64(DataKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.i.MarshalText()
			if err != nil {
				t.Errorf("DataKindType.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("DataKindType.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover DataKindType.String()", func(t *testing.T) {
		const tested = DataKind_FakeLast + 1
		want := "DataKindType(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(DataKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestSchemaKindType_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    SchemaKindType
		want string
	}{
		{name: `0 —> "SchemaKind_null"`,
			k:    SchemaKind_null,
			want: `SchemaKind_null`,
		},
		{name: `1 —> "SchemaKind_GDoc"`,
			k:    SchemaKind_GDoc,
			want: `SchemaKind_GDoc`,
		},
		{name: `SchemaKind_FakeLast —> 17`,
			k:    SchemaKind_FakeLast,
			want: strconv.FormatUint(uint64(SchemaKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("SchemaKindType.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("SchemaKindType.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover SchemaKindType.String()", func(t *testing.T) {
		const tested = SchemaKind_FakeLast + 1
		want := "SchemaKindType(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(SchemaKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestRateLimitKind_String(t *testing.T) {
	tests := []struct {
		name string
		i    RateLimitKind
		want string
	}{
		{name: `0 —> "RateLimitKind_byApp"`,
			i:    RateLimitKind_byApp,
			want: `RateLimitKind_byApp`,
		},
		{name: `1 —> "RateLimitKind_byWorkspace"`,
			i:    RateLimitKind_byWorkspace,
			want: `RateLimitKind_byWorkspace`,
		},
		{name: `RateLimitKind_FakeLast —> "RateLimitKind_FakeLast"`,
			i:    RateLimitKind_FakeLast,
			want: "RateLimitKind_FakeLast",
		},
		{name: `RateLimitKind_FakeLast+1 —> "RateLimitKind(4)"`,
			i:    RateLimitKind_FakeLast + 1,
			want: fmt.Sprintf("RateLimitKind(%d)", RateLimitKind_FakeLast+1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.i.String(); got != tt.want {
				t.Errorf("RateLimitKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceKindType_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ResourceKindType
		want string
	}{
		{name: `0 —> "ResourceKind_null"`,
			k:    ResourceKind_null,
			want: `ResourceKind_null`,
		},
		{name: `1 —> "ResourceKind_CommandFunction"`,
			k:    ResourceKind_CommandFunction,
			want: `ResourceKind_CommandFunction`,
		},
		{name: `ResourceKind_FakeLast —> 3`,
			k:    ResourceKind_FakeLast,
			want: strconv.FormatUint(uint64(ResourceKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("ResourceKindType.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ResourceKindType.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover ResourceKindType.String()", func(t *testing.T) {
		const tested = ResourceKind_FakeLast + 1
		want := "ResourceKindType(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ResourceKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestRateLimitKind_MarshalText(t *testing.T) {
	require := require.New(t)
	for i := 0; i <= int(RateLimitKind_FakeLast); i++ {
		rlk := RateLimitKind(i)
		b, err := rlk.MarshalText()
		require.NoError(err)
		if rlk == RateLimitKind_FakeLast {
			require.Equal(fmt.Sprint(i), string(b))
		} else {
			require.Equal(rlk.String(), string(b))
		}
	}
}

func TestValidatorMatchByQName(t *testing.T) {
	require := require.New(t)
	qn1 := NewQName("test", "n1")
	qn2 := NewQName("test", "n2")
	qn3 := NewQName("test", "n3")

	t.Run("QName only", func(t *testing.T) {
		v := CUDValidator{
			MatchQNames: []QName{qn1, qn2},
		}
		require.True(ValidatorMatchByQName(v, qn1))
		require.True(ValidatorMatchByQName(v, qn2))
		require.False(ValidatorMatchByQName(v, qn3))
	})

	t.Run("func(QName) only", func(t *testing.T) {
		v := CUDValidator{
			MatchFunc: func(qName QName) bool {
				return qName == qn1 || qName == qn2
			},
		}
		require.True(ValidatorMatchByQName(v, qn1))
		require.True(ValidatorMatchByQName(v, qn2))
		require.False(ValidatorMatchByQName(v, qn3))
	})

	t.Run("both func(QName) and QName", func(t *testing.T) {
		v := CUDValidator{
			MatchFunc: func(qName QName) bool {
				return qName == qn1
			},
			MatchQNames: []QName{qn2},
		}
		require.True(ValidatorMatchByQName(v, qn1))
		require.True(ValidatorMatchByQName(v, qn2))
		require.False(ValidatorMatchByQName(v, qn3))
	})
}
