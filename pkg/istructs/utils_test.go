/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package istructs_test

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage_RecordID(t *testing.T) {

	var id istructs.RecordID

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
		wsid := istructs.NewWSID(istructs.NullClusterID+1, istructs.NullWSID+1)
		require.Equal(istructs.WSID(0x0000_8000_0000_0001), wsid)
		require.Equal(istructs.NullClusterID+1, wsid.ClusterID())
		require.Equal(istructs.NullWSID+1, wsid.BaseWSID())
	}

	// Third workspace in second cluster
	{
		wsid := istructs.NewWSID(istructs.NullClusterID+2, istructs.NullWSID+3)
		require.Equal(istructs.WSID(0x0001_0000_0000_0003), wsid)
		require.Equal(istructs.NullClusterID+2, wsid.ClusterID())
		require.Equal(istructs.NullWSID+3, wsid.BaseWSID())
	}

	// Max possible workspace in max possible cluster
	{
		wsid := istructs.NewWSID(istructs.MaxClusterID, 0x7fffffffffff)
		require.Equal(istructs.WSID(0x7fff_ffff_ffff_ffff), wsid)
		require.Equal(istructs.ClusterID(0xffff), wsid.ClusterID())
		require.Equal(istructs.WSID(0x7fffffffffff), wsid.BaseWSID())
	}
}

func TestBaseWSIDOverflow(t *testing.T) {
	istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxBaseWSID)
	require.Panics(t, func() { istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxBaseWSID+1) })
}

// regenerateID: just example for test usage
func regenerateID(id istructs.RecordID) istructs.RecordID {
	const increment = istructs.MaxRawRecordID + 1
	if id.IsRaw() {
		return id + increment
	}
	return id
}

func TestRecordID_IsTemp(t *testing.T) {
	tests := []struct {
		name string
		id   istructs.RecordID
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
				t.Errorf("istructs.RecordID(%v).IsTemp() = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestNullObject(t *testing.T) {
	require := require.New(t)
	builder := istructs.NewNullObjectBuilder()

	require.NotNil(builder)

	require.NotPanics(func() {
		builder.PutInt8("int8", 1)   // #3435 [~server.vsql.smallints/cmp.istructs~impl]
		builder.PutInt16("int16", 1) // #3435 [~server.vsql.smallints/cmp.istructs~impl]
		builder.PutInt32("int32", 1)
		builder.PutInt64("int64", 1)
		builder.PutFloat32("float32", 1)
		builder.PutFloat64("float64", 1)
		builder.PutBytes("bytes", []byte{0})
		builder.PutString("string", "")
		builder.PutQName("QName", appdef.NullQName)
		builder.PutBool("bool", true)
		builder.PutRecordID("istructs.RecordID", istructs.NullRecordID)
		builder.PutNumber("float64", json.Number("1"))
		builder.PutChars("string", "ABC")
		builder.PutNumber("int64", json.Number("1"))

		builder.PutFromJSON(map[string]interface{}{"int32": 1})
		builder.FillFromJSON(map[string]interface{}{"int32": 1, "child": []any{map[string]interface{}{"int32": 1}}})
	})

	require.NotNil(builder.ChildBuilder("child"))

	null, err := builder.Build()
	require.NoError(err)

	require.Nil(null.AsBytes(appdef.NullName))
	require.Equal(float32(0), null.AsFloat32(appdef.NullName))
	require.Equal(float64(0), null.AsFloat64(appdef.NullName))
	require.Zero(null.AsInt8(appdef.NullName))  // #3435 [~server.vsql.smallints/cmp.istructs~impl]
	require.Zero(null.AsInt16(appdef.NullName)) // #3435 [~server.vsql.smallints/cmp.istructs~impl]
	require.Equal(int32(0), null.AsInt32(appdef.NullName))
	require.Equal(int64(0), null.AsInt64(appdef.NullName))
	require.Empty(null.AsString(appdef.NullName))

	require.Equal(appdef.NullQName, null.AsQName(appdef.NullName))
	require.False(null.AsBool(appdef.NullName))
	require.Equal(istructs.NullRecordID, null.AsRecordID(appdef.NullName))

	require.Equal(appdef.NullQName, null.QName())

	// Should not be called
	{
		for range null.SpecifiedValues {
			require.Fail("null.SpecifiedValues should be empty")
		}
		for range null.Containers {
			require.Fail("null.Containers should be empty")
		}
		for range null.Children(appdef.NullName) {
			require.Fail("null.Children should be empty")
		}
		for range null.RecordIDs(true) {
			require.Fail("null.RecordIDs should be empty")
		}
		null.Fields(func(i appdef.IField) bool {
			require.Fail("null.FieldNames should be empty")
			return true
		})
	}

	t.Run("IRecord fields", func(t *testing.T) {
		r := null.AsRecord()
		require.Equal(appdef.NullQName, r.QName())
		require.Equal(appdef.NullQName, r.QName())
		require.Empty(r.Container())
		require.Equal(istructs.NullRecordID, r.ID())
		require.Equal(istructs.NullRecordID, r.Parent())
	})
}

func TestRateLimitKind_String(t *testing.T) {
	tests := []struct {
		name string
		i    istructs.RateLimitKind
		want string
	}{
		{name: `0 —> "RateLimitKind_byApp"`,
			i:    istructs.RateLimitKind_byApp,
			want: `RateLimitKind_byApp`,
		},
		{name: `1 —> "RateLimitKind_byWorkspace"`,
			i:    istructs.RateLimitKind_byWorkspace,
			want: `RateLimitKind_byWorkspace`,
		},
		{name: `RateLimitKind_FakeLast —> "RateLimitKind_FakeLast"`,
			i:    istructs.RateLimitKind_FakeLast,
			want: "RateLimitKind_FakeLast",
		},
		{name: `RateLimitKind_FakeLast+1 —> "RateLimitKind(4)"`,
			i:    istructs.RateLimitKind_FakeLast + 1,
			want: fmt.Sprintf("RateLimitKind(%d)", istructs.RateLimitKind_FakeLast+1),
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
		k    istructs.ResourceKindType
		want string
	}{
		{name: `0 —> "ResourceKind_null"`,
			k:    istructs.ResourceKind_null,
			want: `ResourceKind_null`,
		},
		{name: `1 —> "ResourceKind_CommandFunction"`,
			k:    istructs.ResourceKind_CommandFunction,
			want: `ResourceKind_CommandFunction`,
		},
		{name: `ResourceKind_FakeLast —> 3`,
			k:    istructs.ResourceKind_FakeLast,
			want: utils.UintToString(istructs.ResourceKind_FakeLast),
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
		const tested = istructs.ResourceKind_FakeLast + 1
		want := "ResourceKindType(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ResourceKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestRateLimitKind_MarshalText(t *testing.T) {
	require := require.New(t)
	for i := 0; i <= int(istructs.RateLimitKind_FakeLast); i++ {
		rlk := istructs.RateLimitKind(i)
		b, err := rlk.MarshalText()
		require.NoError(err)
		if rlk == istructs.RateLimitKind_FakeLast {
			require.Equal(strconv.Itoa(i), string(b))
		} else {
			require.Equal(rlk.String(), string(b))
		}
	}
}

func TestUnixMilli(t *testing.T) {
	cases := []struct {
		u istructs.UnixMilli
	}{
		{u: 0}, {u: istructs.UnixMilli(time.Now().UnixMilli())}, {u: math.MaxInt64},
	}
	for _, c := range cases {
		log.Println(c.u.String())
	}
}
