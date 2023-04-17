/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestIsSystemRole(t *testing.T) {
	require := require.New(t)
	require.True(IsSystemRole(QNameRoleProfileOwner))
	require.True(IsSystemRole(QNameRoleSystem))
	require.True(IsSystemRole(QNameRoleWorkspaceAdmin))
	require.True(IsSystemRole(QNameRoleWorkspaceDevice))
	require.True(IsSystemRole(QNameRoleWorkspaceSubject))
	require.False(IsSystemRole(istructs.NewQName(istructs.SysPackage, "test")))
}
