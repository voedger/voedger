/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestIsSystemRole(t *testing.T) {
	require := require.New(t)
	require.True(IsSystemRole(QNameRoleProfileOwner))
	require.True(IsSystemRole(QNameRoleSystem))
	require.True(IsSystemRole(QNameRoleWorkspaceAdmin))
	require.True(IsSystemRole(QNameRoleWorkspaceDevice))
	require.True(IsSystemRole(QNameRoleWorkspaceSubject))
	require.False(IsSystemRole(appdef.NewQName(appdef.SysPackage, "test")))
}
