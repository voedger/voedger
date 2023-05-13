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
		wsid            istructs.WSID
		appWSAmount     istructs.AppWSAmount
		expectedAppWSID istructs.WSID
	}{
		{1, 1, istructs.MaxPseudoBaseWSID + 1},
		{2, 1, istructs.MaxPseudoBaseWSID + 1},
		{3, 1, istructs.MaxPseudoBaseWSID + 1},
		{1, 10, istructs.MaxPseudoBaseWSID + 2},
		{8, 10, istructs.MaxPseudoBaseWSID + 9},
		{10, 10, istructs.MaxPseudoBaseWSID + 1},
		{11, 10, istructs.MaxPseudoBaseWSID + 2},
	}

	for _, c := range cases {
		require.Equal(t, c.expectedAppWSID, GetAppWSID(c.wsid, c.appWSAmount), c)
	}
}

func TestGetPseudoWSID(t *testing.T) {
	fuzz := fuzz.New()
	type src struct {
		entity    string
		clusetrID istructs.ClusterID
	}
	const mask = uint64(0xFFFFFFFFFFFC0000) // ensures that bits 16th-46th are zero
	var srcInstance src
	for i := 0; i < 10000; i++ {
		fuzz.Fuzz(&srcInstance)
		require.Zero(t, uint64(GetPseudoWSID(srcInstance.entity, srcInstance.clusetrID))&mask)
	}
}
