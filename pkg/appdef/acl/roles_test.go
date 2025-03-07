/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestRecursiveRoleAncestors(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	reader := appdef.NewQName("test", "reader")
	writer := appdef.NewQName("test", "writer")
	worker := appdef.NewQName("test", "worker")
	owner := appdef.NewQName("test", "owner")
	admin := appdef.NewQName("test", "adm")

	t.Run("should be ok to build application with roles", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddRole(reader)
		_ = wsb.AddRole(writer)

		_ = wsb.AddRole(worker)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(reader, writer), nil, worker, "grant reader and writer roles to worker")

		_ = wsb.AddRole(owner)
		wsb.GrantAll(
			filter.QNames(worker), owner, "grant worker role to owner")

		_ = wsb.AddRole(admin)
		wsb.GrantAll(
			filter.QNames(owner), admin, "grant owner role to admin")

		app = adb.MustBuild()
	})

	t.Run("test RecursiveRoleAncestors", func(t *testing.T) {
		var tests = []struct {
			role appdef.QName
			want []appdef.QName
		}{
			{reader, []appdef.QName{reader}},
			{writer, []appdef.QName{writer}},
			{worker, []appdef.QName{worker, reader, writer}},
			{owner, []appdef.QName{owner, worker, reader, writer}},
			{admin, []appdef.QName{admin, owner, worker, reader, writer}},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.role.String(), func(t *testing.T) {
				roles := acl.RecursiveRoleAncestors(appdef.Role(app.Type, tt.role), ws)
				require.ElementsMatch(tt.want, roles)
			})
		}
	})
}

// #3127: GRANT ROLE TO ROLE from ancestor WS
func TestRecursiveRoleAncestorsWSInheritance(t *testing.T) {
	// ABSTRACT WORKSPACE AbstractWS (
	// 	ROLE AWSRole;
	// );

	// ABSTRACT WORKSPACE WS_1 INHERITS AbstractWS (
	// 	ROLE R1;
	// 	GRANT R1 TO AWSRole;
	// );

	// ABSTRACT WORKSPACE WS_2 INHERITS AbstractWS (
	// 	ROLE R2;
	// 	GRANT R2 TO AWSRole;
	// );

	// WORKSPACE WS_3 INHERITS WS_1, WS_2 (
	// 	ROLE R3;
	// 	GRANT R3 TO AWSRole;
	// );

	require := require.New(t)

	var app appdef.IAppDef

	awsName := appdef.NewQName("test", "abstractWS")
	awsRole := appdef.NewQName("test", "awsRole")
	wsName := []appdef.QName{appdef.NewQName("test", "ws_1"), appdef.NewQName("test", "ws_2"), appdef.NewQName("test", "ws_3")}
	wsRole := []appdef.QName{appdef.NewQName("test", "r_1"), appdef.NewQName("test", "r_2"), appdef.NewQName("test", "r_3")}

	t.Run("should be ok to build application with roles", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		aws := adb.AddWorkspace(awsName)
		_ = aws.AddRole(awsRole)

		ws1 := adb.AddWorkspace(wsName[0])
		ws1.SetAncestors(awsName)
		_ = ws1.AddRole(wsRole[0])
		ws1.GrantAll(filter.QNames(wsRole[0]), awsRole, "grant r1 to awsRole")

		ws2 := adb.AddWorkspace(wsName[1])
		ws2.SetAncestors(awsName)
		_ = ws2.AddRole(wsRole[1])
		ws2.GrantAll(filter.QNames(wsRole[1]), awsRole, "grant r2 to awsRole")

		ws3 := adb.AddWorkspace(wsName[2])
		ws3.SetAncestors(wsName[0], wsName[1])
		_ = ws3.AddRole(wsRole[2])
		ws3.GrantAll(filter.QNames(wsRole[2]), awsRole, "grant r3 to awsRole")

		app = adb.MustBuild()
	})

	t.Run("test RecursiveRoleAncestors", func(t *testing.T) {
		var tests = []struct {
			role appdef.QName
			ws   appdef.QName
			want []appdef.QName
		}{
			{awsRole, awsName, []appdef.QName{awsRole}},
			{awsRole, wsName[0], []appdef.QName{awsRole, wsRole[0]}},
			{awsRole, wsName[1], []appdef.QName{awsRole, wsRole[1]}},
			{awsRole, wsName[2], []appdef.QName{awsRole, wsRole[0], wsRole[1], wsRole[2]}},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s in %s", tt.role.String(), tt.ws.String()), func(t *testing.T) {
				ws := app.Workspace(tt.ws)
				role := appdef.Role(ws.Type, tt.role)
				roles := acl.RecursiveRoleAncestors(role, ws)
				require.ElementsMatch(tt.want, roles)
			})
		}
	})
}
