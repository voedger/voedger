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
	"github.com/voedger/voedger/pkg/appdef"
)

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
	require.NoError(err)
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
		require.NoError(err)

		// Unmarshal

		var appqname2 = AppQName{}
		err = json.Unmarshal(j, &appqname2)
		require.NoError(err)

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
		require.NoError(err)

		// Unmarshal

		var ms2 = myStruct{}
		err = json.Unmarshal(j, &ms2)
		require.NoError(err)

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

func TestAppQName_Compare(t *testing.T) {
	require := require.New(t)

	q1_1 := NewAppQName("sys", "registry")
	q1_2 := NewAppQName("sys", "registry")
	require.Equal(q1_1, q1_2)
	require.True(q1_1 == q1_2)

	q2 := appdef.NewQName("sys", "registry2")
	require.NotEqual(q1_1, q2)
}

func TestAppQName_Json_NullQName(t *testing.T) {

	require := require.New(t)
	t.Run("Marshall/unmarshall NullQName", func(t *testing.T) {

		aqn := NullAppQName

		// Marshal

		j, err := json.Marshal(&aqn)
		require.NoError(err)

		// Unmarshal

		var aqn2 = AppQName{}
		err = json.Unmarshal(j, &aqn2)
		require.NoError(err)

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

	require.Nil(null.AsBytes(appdef.NullName))
	require.Equal(float32(0), null.AsFloat32(appdef.NullName))
	require.Equal(float64(0), null.AsFloat64(appdef.NullName))
	require.Equal(int32(0), null.AsInt32(appdef.NullName))
	require.Equal(int64(0), null.AsInt64(appdef.NullName))
	require.Equal("", null.AsString(appdef.NullName))

	require.Equal(appdef.NullQName, null.AsQName(appdef.NullName))
	require.Equal(false, null.AsBool(appdef.NullName))
	require.Equal(NullRecordID, null.AsRecordID(appdef.NullName))

	require.Equal(appdef.NullQName, null.QName())

	// Should not be called
	{
		null.Containers(nil)
		null.Elements(appdef.NullName, nil)
		null.RecordIDs(true, nil)
		null.FieldNames(nil)
	}

	t.Run("IRecord fields", func(t *testing.T) {
		r := null.AsRecord()
		require.Equal(appdef.NullQName, r.QName())
		require.Equal(appdef.NullQName, r.QName())
		require.Equal("", r.Container())
		require.Equal(NullRecordID, r.ID())
		require.Equal(NullRecordID, r.Parent())

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
	qn1 := appdef.NewQName("test", "n1")
	qn2 := appdef.NewQName("test", "n2")
	qn3 := appdef.NewQName("test", "n3")
	expectedWSID := WSID(42)

	t.Run("QName only", func(t *testing.T) {
		v := CUDValidator{
			MatchQNames: []appdef.QName{qn1, qn2},
		}
		require.True(ValidatorMatchByQName(v, qn1, expectedWSID, qn1))
		require.True(ValidatorMatchByQName(v, qn2, expectedWSID, qn1))
		require.False(ValidatorMatchByQName(v, qn3, expectedWSID, qn1))
	})

	t.Run("func(QName) only", func(t *testing.T) {
		v := CUDValidator{
			MatchFunc: func(qName appdef.QName, wsid WSID, cmdQName appdef.QName) bool {
				return (qName == qn1 || qName == qn2) && wsid == expectedWSID && cmdQName == qn1
			},
		}
		require.True(ValidatorMatchByQName(v, qn1, expectedWSID, qn1))
		require.True(ValidatorMatchByQName(v, qn2, expectedWSID, qn1))
		require.False(ValidatorMatchByQName(v, qn3, expectedWSID, qn1))
	})

	t.Run("both func(QName) and QName", func(t *testing.T) {
		v := CUDValidator{
			MatchFunc: func(qName appdef.QName, wsid WSID, cmdQName appdef.QName) bool {
				return qName == qn1 && wsid == expectedWSID && cmdQName == qn1
			},
			MatchQNames: []appdef.QName{qn2},
		}
		require.True(ValidatorMatchByQName(v, qn1, expectedWSID, qn1))
		require.True(ValidatorMatchByQName(v, qn2, expectedWSID, qn1))
		require.False(ValidatorMatchByQName(v, qn3, expectedWSID, qn1))
	})
}
