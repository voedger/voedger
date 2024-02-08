/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package istructs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterApps(t *testing.T) {

	require := require.New(t)

	t.Run("All apps has IDs", func(t *testing.T) {
		require.Len(ClusterApps, int(ClusterAppID_FakeLast))
	})

	t.Run("All IDs are unique", func(t *testing.T) {
		vals := map[ClusterAppID]AppQName{}
		for k, v := range ClusterApps {
			vals[v] = k
		}
		require.Len(vals, int(ClusterAppID_FakeLast))
	})
}

func TestMainCluster(t *testing.T) {
	require := require.New(t)
	require.Equal(MainClusterID, ClusterID(1))
}

func TestWSID(t *testing.T) {
	require := require.New(t)
	require.Equal(MaxPseudoBaseWSID, WSID(0xffff))
	require.Equal(FirstBaseAppWSID, WSID(0xffff+1))
	require.Equal(FirstBaseUserWSID, WSID(0xffff+0xffff+1))
}
