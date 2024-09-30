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

func TestBasicUsage_GetAppWSID(t *testing.T) {
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
		require.Equal(t, c.expectedAppWSID, GetAppWSID(c.wsid, c.numAppWorkspaces), c)
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

func TestAppWSIDToPseudoWSID(t *testing.T) {
	numAppWorkspaces := istructs.NumAppWorkspaces(10)
	pseudoWSIDInitial := GetPseudoWSID(istructs.NullWSID, "test", istructs.CurrentClusterID())
	appWSIDExpected := GetAppWSID(pseudoWSIDInitial, numAppWorkspaces)

	// could be any but must lead to the initial appWSIDExpected
	psuedoWSID_someNew := AppWSIDToPseudoWSID(appWSIDExpected)

	appWSIDActual := GetAppWSID(psuedoWSID_someNew, numAppWorkspaces)
	require.Equal(t, appWSIDExpected, appWSIDActual)
}
