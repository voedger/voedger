/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestPseudoWSIDToAppWSID(t *testing.T) {
	cases := []struct {
		wsid             istructs.WSID
		numAppWorkspaces istructs.NumAppWorkspaces
		expectedAppWSID  istructs.WSID
	}{
		{1, 1, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+1)},
		{2, 1, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+1)},
		{3, 1, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+1)},
		{1, 10, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+2)},
		{8, 10, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+9)},
		{10, 10, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+1)},
		{11, 10, istructs.NewWSID(istructs.CurrentClusterID(), istructs.MaxPseudoBaseWSID+2)},
	}

	for _, c := range cases {
		require.Equal(t, c.expectedAppWSID, PseudoWSIDToAppWSID(c.wsid, c.numAppWorkspaces), c)
	}
}

func TestGetPseudoWSID(t *testing.T) {
	fuzz := fuzz.New()
	type src struct {
		entity    string
		clusterID istructs.ClusterID
	}
	const mask = uint64(0xFFFFFFFFFFFC0000)
	var srcInstance src
	for i := 0; i < 10000; i++ {
		fuzz.Fuzz(&srcInstance)
		require.Zero(t, uint64(GetPseudoWSID(istructs.NullWSID, srcInstance.entity, srcInstance.clusterID))&mask)
		require.Zero(t, uint64(GetPseudoWSID(istructs.NullWSID+1, srcInstance.entity, srcInstance.clusterID))&mask)
	}
}

func TestAppWSNumber(t *testing.T) {
	tests := []struct {
		offset           istructs.WSID
		numAppWorkspaces istructs.NumAppWorkspaces
		expectedNumber   uint32
	}{
		{0, 10, 6},  // 65536 % 10 = 6
		{7, 5, 3},   // (65536+7) % 5 = 3
		{10, 10, 6}, // (65536+10) % 10 = 6 (wrap around)
		{100, 1, 0}, // Any WSID % 1 = 0 (single workspace)
	}

	for i, tt := range tests {
		wsid := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+tt.offset)
		result := AppWSNumber(wsid, tt.numAppWorkspaces)
		require.Equal(t, tt.expectedNumber, result, "test case %d", i)
	}

	t.Run("maximum workspaces", func(t *testing.T) {
		// Test with maximum allowed number of app workspaces
		maxWorkspaces := istructs.NumAppWorkspaces(istructs.MaxNumAppWorkspaces)
		wsid := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID)
		result := AppWSNumber(wsid, maxWorkspaces)
		require.Equal(t, uint32(0), result) // 65536 % 32768 = 0
	})

	t.Run("cluster ID independence", func(t *testing.T) {
		// Should work the same regardless of cluster ID
		baseWSID := istructs.FirstBaseAppWSID + 7
		expected := uint32(3) // (65536+7) % 5 = 3

		wsid1 := istructs.NewWSID(istructs.ClusterID(1), baseWSID)
		wsid2 := istructs.NewWSID(istructs.ClusterID(10), baseWSID)

		require.Equal(t, expected, AppWSNumber(wsid1, 5))
		require.Equal(t, expected, AppWSNumber(wsid2, 5))
	})
}
