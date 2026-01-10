/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package istructs

import (
	"log"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

//nolint:unconvert
const (
	_ = uint16(QNameIDForError - 1)
	_ = uint16(1 - QNameIDForError)
	_ = uint16(QNameIDCommandCUD - 2)
	_ = uint16(2 - QNameIDCommandCUD)
	_ = uint16(QNameIDForCorruptedData - 3)
	_ = uint16(3 - QNameIDForCorruptedData)
	_ = uint16(QNameIDWLogOffsetSequence - 4)
	_ = uint16(4 - QNameIDWLogOffsetSequence)
	_ = uint16(QNameIDRecordIDSequence - 5)
	_ = uint16(5 - QNameIDRecordIDSequence)

	_ = uint16(QNameIDSysLast - 0xFF)
	_ = uint16(0xFF - QNameIDSysLast)
)

func TestConst(t *testing.T) {
	exp := NewWSID(math.MaxUint16, WSID(MaxBaseWSID))
	act := MaxAllowedWSID
	p := uint64(math.Pow(2, 63)) - 1
	log.Printf("%64b\n", exp)
	log.Printf("%64b\n", act)
	log.Printf("%64b\n", p)
}

func TestClusterApps(t *testing.T) {

	require := require.New(t)

	t.Run("All apps has IDs", func(t *testing.T) {
		require.Len(ClusterApps, int(ClusterAppID_FakeLast))
	})

	t.Run("All IDs are unique", func(t *testing.T) {
		vals := map[ClusterAppID]appdef.AppQName{}
		for k, v := range ClusterApps {
			vals[v] = k
		}
		require.Len(vals, int(ClusterAppID_FakeLast))
	})
}

func TestMainCluster(t *testing.T) {
	require := require.New(t)
	require.Equal(MainClusterID_useWithCare, ClusterID(1))
	require.Equal(CurrentClusterID(), ClusterID(1))
}

func TestWSID(t *testing.T) {
	require := require.New(t)
	require.Equal(MaxPseudoBaseWSID, WSID(0xffff))
	require.Equal(FirstBaseAppWSID, WSID(0xffff+1))
	require.Equal(FirstBaseUserWSID, WSID(0xffff+0xffff+1))

	require.Equal(FirstReservedWSID, WSID(0xffff+1+0x8000))
	require.Equal(FirstReservedWSID, WSID(98304))
	require.Equal(FirstReservedWSID, WSID(0x18000))
	require.Equal(GuestWSID, WSID(0x18000))
	require.Equal(GuestWSID, WSID(98304))

	require.Equal(FirstPseudoBaseWSID+MaxPseudoBaseWSID+1, FirstBaseAppWSID)
	require.Equal(FirstBaseAppWSID+MaxNumAppWorkspaces, FirstReservedWSID)
	require.Equal(FirstReservedWSID+NumReservedWSID, FirstBaseUserWSID)
}
