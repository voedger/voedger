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
	require.False(IsSystemRole(appdef.NewQName(appdef.SysPackage, "test")))
}

func TestRolesInheritance(t *testing.T) {
	for qName := range rolesInheritance {
		require.NotEqual(t, appdef.NullQName, QNameAncestor(qName))
	}
	require.Equal(t, appdef.NullQName, QNameAncestor(appdef.NewQName("missing", "missing")))
}

func TestIsSystemPrincipal(t *testing.T) {
	t.Fatal()
}
