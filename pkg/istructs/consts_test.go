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
		require.Equal(int(ClusterAppID_FakeLast), len(ClusterApps))
	})

	t.Run("All IDs are unique", func(t *testing.T) {
		vals := map[ClusterAppID]AppQName{}
		for k, v := range ClusterApps {
			vals[v] = k
		}
		require.Equal(int(ClusterAppID_FakeLast), len(vals))
	})
}

func TestMainCluster(t *testing.T) {
	require := require.New(t)
	require.Equal(ClusterID(1), MainClusterID)
}

func TestWSID(t *testing.T) {
	require := require.New(t)
	require.Equal(WSID(0xffff), MaxPseudoBaseWSID)
	require.Equal(WSID(0xffff+1), FirstBaseAppWSID)
	require.Equal(WSID(0xffff+0xffff+1), FirstBaseUserWSID)
}
