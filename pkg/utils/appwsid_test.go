/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

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
