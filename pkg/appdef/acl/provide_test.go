/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl_test

import (
	"fmt"
	"testing"

	"maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/set"
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

func TestIsOperationAllowed(t *testing.T) {
	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	cDocName := appdef.NewQName("test", "cDoc")
	oDocName := appdef.NewQName("test", "oDoc")
	viewName := appdef.NewQName("test", "view")
	queryName := appdef.NewQName("test", "qry")
	cmdName := appdef.NewQName("test", "cmd")
	tagName := appdef.NewQName("test", "tag")

	everyone := appdef.NewQName("test", "everyone")
	reader := appdef.NewQName("test", "reader")
	writer := appdef.NewQName("test", "writer")
	admin := appdef.NewQName("test", "admin")
	intruder := appdef.NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagName)

		cDoc := wsb.AddCDoc(cDocName)
		cDoc.
			AddField("field1", appdef.DataKind_int32, true).
			AddField("hiddenField", appdef.DataKind_int32, false).
			AddField("field3", appdef.DataKind_int32, false)
		cDoc.SetTag(tagName)

		oDoc := wsb.AddODoc(oDocName)
		oDoc.AddField("field1", appdef.DataKind_int32, true)
		oDoc.SetTag(tagName)

		view := wsb.AddView(viewName)
		view.Key().PartKey().AddField("field1", appdef.DataKind_int32)
		view.Key().ClustCols().AddField("field2", appdef.DataKind_int32)
		view.Value().AddField("field3", appdef.DataKind_int32, false)

		qry := wsb.AddQuery(queryName)
		qry.SetResult(cDocName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetParam(appdef.QNameANY)

		_ = wsb.AddRole(everyone)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(viewName),
			nil,
			everyone,
			"grant select view to everyone")

		_ = wsb.AddRole(reader)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			reader,
			"grant inherits everyone to reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			reader,
			"grant select any CDoc with tag to reader")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(cDocName),
			[]appdef.FieldName{"hiddenField"},
			reader,
			"revoke select doc.field1 from reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(queryName),
			nil,
			reader,
			"grant execute query to reader")

		_ = wsb.AddRole(writer)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			writer,
			"grant inherits everyone to writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Insert},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			writer,
			"grant insert any CDoc with tag to writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(cDocName),
			[]appdef.FieldName{"field1", "hiddenField", "field3"},
			writer,
			"grant update doc.field[1,hidden,3] to writer")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(cDocName),
			[]appdef.FieldName{"hiddenField"},
			writer,
			"revoke update doc.hiddenField from writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.AllWSFunctions(wsName),
			nil,
			writer,
			"grant execute all commands and queries to writer")

		_ = wsb.AddRole(admin)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			admin,
			"grant inherits everyone to admin")
		wsb.GrantAll(filter.AllWSTables(wsName), admin)
		wsb.GrantAll(filter.AllWSFunctions(wsName), admin)

		_ = wsb.AddRole(intruder)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			intruder,
			"grant inherits everyone to intruder")
		wsb.RevokeAll(
			filter.WSTypes(wsName, appdef.TypeKind_CDoc),
			intruder,
			"revoke all access to CDocs from intruder")
		wsb.RevokeAll(
			filter.AllWSFunctions(wsName),
			intruder,
			"revoke all access to functions from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("test IsOperationAllowed", func(t *testing.T) {
		var tests = []struct {
			name    string
			op      appdef.OperationKind
			res     appdef.QName
			fields  []appdef.FieldName
			role    appdef.QName
			allowed bool
		}{
			// iauthnz.QNameRoleSystem test
			{
				name:    "allow select * from doc for system",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  nil,
				role:    appdef.QNameRoleSystem,
				allowed: true,
			},
			{
				name:    "allow insert to doc for system",
				op:      appdef.OperationKind_Insert,
				res:     cDocName,
				fields:  nil,
				role:    appdef.QNameRoleSystem,
				allowed: true,
			},
			{
				name:    "allow update doc for system",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  nil,
				role:    appdef.QNameRoleSystem,
				allowed: true,
			},
			{
				name:    "allow select ? from view for system",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    appdef.QNameRoleSystem,
				allowed: true,
			},
			// everyone test
			{
				name:    "allow select ? from view for everyone",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			// reader tests
			{
				name:    "allow select doc.field1 for reader",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  []appdef.FieldName{"field1"},
				role:    reader,
				allowed: true,
			},
			{
				name:    "deny select doc.hiddenField for reader",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  []appdef.FieldName{"hiddenField"},
				role:    reader,
				allowed: false,
			},
			{
				name:    "allow select ? from doc for reader",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  nil,
				role:    reader,
				allowed: true,
			},
			{
				name:    "allow select ? from view for reader",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    reader,
				allowed: true,
			},
			{
				name:    "deny select * from doc for reader",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  []appdef.FieldName{"field1", "hiddenField", "field3"},
				role:    reader,
				allowed: false,
			},
			{
				name:    "deny insert doc for reader",
				op:      appdef.OperationKind_Insert,
				res:     cDocName,
				fields:  nil,
				role:    reader,
				allowed: false,
			},
			{
				name:    "deny update doc for reader",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  nil,
				role:    reader,
				allowed: false,
			},
			{
				name:    "allow execute query for reader",
				op:      appdef.OperationKind_Execute,
				res:     queryName,
				fields:  nil,
				role:    reader,
				allowed: true,
			},
			{
				name:    "deny execute cmd for reader",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				fields:  nil,
				role:    reader,
				allowed: false,
			},
			// writer tests
			{
				name:    "deny select ? from doc for writer",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  nil,
				role:    writer,
				allowed: false,
			},
			{
				name:    "allow insert to doc for writer",
				op:      appdef.OperationKind_Insert,
				res:     cDocName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			{
				name:    "allow update doc for writer",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			{
				name:    "allow select ? from view for writer",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			{
				name:    "deny update doc.hiddenField for writer",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  []appdef.FieldName{"hiddenField"},
				role:    writer,
				allowed: false,
			},
			{ // #3148: appparts: ACTIVATE/DEACTIVATE in IsOperationAllowed
				name:    "deny deactivate doc for writer",
				op:      appdef.OperationKind_Deactivate,
				res:     cDocName,
				fields:  nil,
				role:    writer,
				allowed: false,
			},
			{
				name:    "allow execute cmd for writer",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			{
				name:    "allow execute query for writer",
				op:      appdef.OperationKind_Execute,
				res:     queryName,
				fields:  nil,
				role:    writer,
				allowed: true,
			},
			// admin tests
			{
				name:    "allow select ? from doc for admin",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			{
				name:    "allow insert to doc for admin",
				op:      appdef.OperationKind_Insert,
				res:     cDocName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			{
				name:    "allow update doc for admin",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			{ // #3148: appparts: ACTIVATE/DEACTIVATE in IsOperationAllowed
				name:    "allow deactivate doc for admin",
				op:      appdef.OperationKind_Deactivate,
				res:     cDocName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			{
				name:    "allow execute cmd for admin",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			{
				name:    "allow select ? from view for admin",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    admin,
				allowed: true,
			},
			// intruder tests
			{
				name:    "allow select ? from view for intruder",
				op:      appdef.OperationKind_Select,
				res:     viewName,
				fields:  nil,
				role:    intruder,
				allowed: true,
			},
			{
				name:    "deny select ? from doc for intruder",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				fields:  nil,
				role:    intruder,
				allowed: false,
			},
			{
				name:    "deny insert doc for intruder",
				op:      appdef.OperationKind_Insert,
				res:     cDocName,
				fields:  nil,
				role:    intruder,
				allowed: false,
			},
			{
				name:    "deny update doc for intruder",
				op:      appdef.OperationKind_Update,
				res:     cDocName,
				fields:  nil,
				role:    intruder,
				allowed: false,
			},
			{
				name:    "deny execute query for intruder",
				op:      appdef.OperationKind_Execute,
				res:     queryName,
				fields:  nil,
				role:    intruder,
				allowed: false,
			},
			{
				name:    "deny execute cmd for intruder",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				fields:  nil,
				role:    intruder,
				allowed: false,
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, []appdef.QName{tt.role})
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
			})
		}
	})

	t.Run("test IsOperationAllowed with multiple roles", func(t *testing.T) {
		var tests = []struct {
			name    string
			op      appdef.OperationKind
			res     appdef.QName
			fields  []appdef.FieldName
			role    []appdef.QName
			allowed bool
		}{
			{
				name:    "allow select doc for [reader, writer]",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				role:    []appdef.QName{reader, writer},
				allowed: true,
			},
			{
				name:    "deny select doc for [reader, intruder]",
				op:      appdef.OperationKind_Select,
				res:     cDocName,
				role:    []appdef.QName{reader, intruder},
				allowed: false,
			},
			{
				name:    "allow execute cmd for [reader, writer]",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				role:    []appdef.QName{reader, writer},
				allowed: true,
			},
			{
				name:    "deny execute cmd for [writer, intruder]",
				op:      appdef.OperationKind_Execute,
				res:     cmdName,
				role:    []appdef.QName{writer, intruder},
				allowed: false,
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, tt.role)
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
			})
		}
	})

	t.Run("test IsOperationAllowed with errors", func(t *testing.T) {
		var tests = []struct {
			name   string
			op     appdef.OperationKind
			res    appdef.QName
			fields []appdef.FieldName
			role   []appdef.QName
			error  error
			errHas any
		}{
			{
				name:   "unsupported operation",
				op:     appdef.OperationKind_Inherits,
				res:    cDocName,
				role:   []appdef.QName{reader},
				error:  appdef.ErrUnsupportedError,
				errHas: "Inherits",
			},
			{
				name:   "resource not found",
				op:     appdef.OperationKind_Insert,
				res:    appdef.NewQName("test", "unknown"),
				role:   []appdef.QName{reader},
				error:  appdef.ErrNotFoundError,
				errHas: "test.unknown",
			},
			{
				name:   "not a structure",
				op:     appdef.OperationKind_Select,
				res:    cmdName,
				role:   []appdef.QName{reader},
				error:  appdef.ErrIncompatibleError,
				errHas: cmdName,
			},
			{ // #3148: appparts: ACTIVATE/DEACTIVATE in IsOperationAllowed
				name:   "not a record",
				op:     appdef.OperationKind_Deactivate,
				res:    queryName,
				role:   []appdef.QName{admin},
				error:  appdef.ErrIncompatibleError,
				errHas: queryName,
			},
			{ // #3148: appparts: ACTIVATE/DEACTIVATE in IsOperationAllowed
				name:   "has not sys.Active field",
				op:     appdef.OperationKind_Deactivate,
				res:    oDocName,
				role:   []appdef.QName{admin},
				error:  appdef.ErrNotFoundError,
				errHas: appdef.SystemField_IsActive,
			},
			{
				name:   "not a function",
				op:     appdef.OperationKind_Execute,
				res:    cDocName,
				role:   []appdef.QName{writer},
				error:  appdef.ErrIncompatibleError,
				errHas: cDocName,
			},
			{
				name:   "field not found",
				op:     appdef.OperationKind_Update,
				res:    cDocName,
				fields: []appdef.FieldName{"unknown"},
				role:   []appdef.QName{writer},
				error:  appdef.ErrNotFoundError,
				errHas: "unknown",
			},
			{
				name:   "no participants",
				op:     appdef.OperationKind_Execute,
				res:    cmdName,
				error:  appdef.ErrMissedError,
				errHas: "participant",
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, tt.role)
				require.Error(err, require.Is(tt.error), require.Has(tt.errHas))
				require.False(allowed)
			})
		}
	})
}

func TestIsOperationAllowedWSInheritances(t *testing.T) {
	// #3113: IsOperationAllowed should use request workspace
	// #3127: GRANT ROLE TO ROLE from ancestor WS
	//
	// # Test plan:
	// ┌─────────────────────────┬─────────────┐
	// │       workspaces        │  resources  │
	// ├───────────┬─────────────┼──────┬──────┤
	// │ ancestor  │ descendants │ doc1 │ doc2 │
	// ├───────────┴─────────────┼──────┼──────┤
	// │ abstractWS              │ S--  │ S--  │ grant select all tables to role
	// │     ▲     ┌─◄─ ws1      │ SIU  │ S--  │ grant all on doc1 to role
	// │     └─————┼─◄─ ws2      │ S--  │ SIU  │ grant all on doc2 to r2, grant r2 to role (#3127)
	// │           └─◄─ ws3      │ ---  │ ---  │ revoke all on doc1, doc2 from role
	// └─────────────────────────┴──────┴──────┘

	require := require.New(t)

	var app appdef.IAppDef

	awsName := appdef.NewQName("test", "abstractWS")
	ws1Name := appdef.NewQName("test", "ws1")
	ws2Name := appdef.NewQName("test", "ws2")
	ws3Name := appdef.NewQName("test", "ws3")

	roleName := appdef.NewQName("test", "role")
	r2Name := appdef.NewQName("test", "r2")

	doc1Name := appdef.NewQName("test", "doc1")
	doc2Name := appdef.NewQName("test", "doc2")
	docs := []appdef.QName{doc1Name, doc2Name}

	t.Run("should be ok to build application", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		aws := adb.AddWorkspace(awsName)
		_ = aws.AddCDoc(doc1Name)
		_ = aws.AddCDoc(doc2Name)
		_ = aws.AddRole(roleName)
		aws.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, filter.AllWSTables(awsName), nil, roleName, "grant select all tables to role")

		ws1 := adb.AddWorkspace(ws1Name)
		ws1.SetAncestors(awsName)
		ws1.GrantAll(filter.QNames(doc1Name), roleName, "grant all on doc1 to role")

		ws2 := adb.AddWorkspace(ws2Name)
		ws2.SetAncestors(awsName)
		_ = ws2.AddRole(r2Name)
		ws2.GrantAll(filter.QNames(doc2Name), r2Name, "grant all on doc2 to r2")
		ws2.GrantAll(filter.WSTypes(ws2Name, appdef.TypeKind_Role), roleName, "grant {ALL WS ROLES} to role") // #3127

		ws3 := adb.AddWorkspace(ws3Name)
		ws3.SetAncestors(awsName)
		ws3.RevokeAll(filter.QNames(doc1Name, doc2Name), roleName, "revoke all on doc1, doc2 from role")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	require.NotNil(app)

	t.Run("test IsOperationAllowed", func(t *testing.T) {
		var tests = []struct {
			ws      appdef.QName
			allowed map[appdef.QName]appdef.OperationsSet
		}{
			{
				ws: awsName,
				allowed: map[appdef.QName]appdef.OperationsSet{
					doc1Name: set.From(appdef.OperationKind_Select),
					doc2Name: set.From(appdef.OperationKind_Select),
				},
			},
			{
				ws: ws1Name,
				allowed: map[appdef.QName]appdef.OperationsSet{
					doc1Name: set.From(appdef.OperationKind_Select, appdef.OperationKind_Insert, appdef.OperationKind_Update),
					doc2Name: set.From(appdef.OperationKind_Select),
				},
			},
			{
				ws: ws2Name,
				allowed: map[appdef.QName]appdef.OperationsSet{
					doc1Name: set.From(appdef.OperationKind_Select),
					doc2Name: set.From(appdef.OperationKind_Select, appdef.OperationKind_Insert, appdef.OperationKind_Update),
				},
			},
			{
				ws:      ws3Name,
				allowed: map[appdef.QName]appdef.OperationsSet{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.ws.Entity(), func(t *testing.T) {
				ws := app.Workspace(tt.ws)
				ops := []appdef.OperationKind{appdef.OperationKind_Select, appdef.OperationKind_Insert, appdef.OperationKind_Update}
				for _, op := range ops {
					for _, doc := range docs {
						want := tt.allowed[doc].Contains(op)
						got, err := acl.IsOperationAllowed(ws, op, doc, nil, []appdef.QName{roleName})
						require.NoError(err)
						require.Equal(want, got, "%s %s", doc, op)
					}
				}
			})
		}
	})
}

// [~server.apiv2.role/cmp.publishedTypes~test]
func TestPublishedTypes(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	cDocName := appdef.NewQName("test", "cDoc")
	oDocName := appdef.NewQName("test", "oDoc")
	viewName := appdef.NewQName("test", "view")
	queryName := appdef.NewQName("test", "qry")
	cmdName := appdef.NewQName("test", "cmd")
	tagName := appdef.NewQName("test", "tag")

	everyone := appdef.NewQName("test", "everyone") // can select from view
	reader := appdef.NewQName("test", "reader")     // everyone + select from CDoc without hiddenfield + execute query
	writer := appdef.NewQName("test", "writer")     // everyone + insert CDoc + update CDoc without hiddenfield + execute all commands and queries
	admin := appdef.NewQName("test", "admin")       // everyone + all access to all tables and functions
	intruder := appdef.NewQName("test", "intruder") // everyone - all access to all tables and functions

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagName)

		cDoc := wsb.AddCDoc(cDocName)
		cDoc.
			AddField("field1", appdef.DataKind_int32, true).
			AddField("hiddenField", appdef.DataKind_int32, false).
			AddField("field3", appdef.DataKind_int32, false)
		cDoc.SetTag(tagName)

		oDoc := wsb.AddODoc(oDocName)
		oDoc.AddField("field1", appdef.DataKind_int32, true)
		oDoc.SetTag(tagName)

		view := wsb.AddView(viewName)
		view.Key().PartKey().AddField("field1", appdef.DataKind_int32)
		view.Key().ClustCols().AddField("field2", appdef.DataKind_int32)
		view.Value().AddField("field3", appdef.DataKind_int32, false)

		qry := wsb.AddQuery(queryName)
		qry.SetResult(cDocName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetParam(appdef.QNameANY)

		_ = wsb.AddRole(everyone)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(viewName),
			nil,
			everyone,
			"grant select view to everyone")

		_ = wsb.AddRole(reader)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			reader,
			"grant inherits everyone to reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			reader,
			"grant select any CDoc with tag to reader")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(cDocName),
			[]appdef.FieldName{"hiddenField"},
			reader,
			"revoke select doc.field1 from reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(queryName),
			nil,
			reader,
			"grant execute query to reader")

		_ = wsb.AddRole(writer)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			writer,
			"grant inherits everyone to writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Insert},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			writer,
			"grant insert any CDoc with tag to writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(cDocName),
			[]appdef.FieldName{"field1", "hiddenField", "field3"},
			writer,
			"grant update doc.field[1,hidden,3] to writer")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(cDocName),
			[]appdef.FieldName{"hiddenField"},
			writer,
			"revoke update doc.hiddenField from writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.AllWSFunctions(wsName),
			nil,
			writer,
			"grant execute all commands and queries to writer")

		_ = wsb.AddRole(admin)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			admin,
			"grant inherits everyone to admin")
		wsb.GrantAll(filter.AllWSTables(wsName), admin)
		wsb.GrantAll(filter.AllWSFunctions(wsName), admin)

		_ = wsb.AddRole(intruder)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(everyone),
			nil,
			intruder,
			"grant inherits everyone to intruder")
		wsb.RevokeAll(
			filter.WSTypes(wsName, appdef.TypeKind_CDoc),
			intruder,
			"revoke all access to CDocs from intruder")
		wsb.RevokeAll(
			filter.AllWSFunctions(wsName),
			intruder,
			"revoke all access to functions from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("test PublishedTypes", func(t *testing.T) {
		var tests = []struct {
			role appdef.QName
			want map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName
		}{
			// iauthnz.QNameRoleSystem test
			{
				role: appdef.QNameRoleSystem,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					cDocName: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
					oDocName: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
					viewName: {
						appdef.OperationKind_Select: nil,
						appdef.OperationKind_Insert: nil,
						appdef.OperationKind_Update: nil,
					},
					queryName: {
						appdef.OperationKind_Execute: nil,
					},
					cmdName: {
						appdef.OperationKind_Execute: nil,
					},
				},
			},
			// everyone test
			{
				role: everyone,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					viewName: {
						appdef.OperationKind_Select: nil,
					},
				},
			},
			// reader test
			{
				role: reader,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					viewName: {
						appdef.OperationKind_Select: nil,
					},
					cDocName: {
						appdef.OperationKind_Select: {
							appdef.SystemField_QName,
							appdef.SystemField_ID,
							appdef.SystemField_IsActive,
							appdef.FieldName("field1"),
							appdef.FieldName("field3"),
						},
					},
					queryName: {
						appdef.OperationKind_Execute: nil,
					},
				},
			},
			// writer test
			{
				role: writer,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					viewName: {
						appdef.OperationKind_Select: nil,
					},
					cDocName: {
						appdef.OperationKind_Insert: nil,
						appdef.OperationKind_Update: {
							appdef.FieldName("field1"),
							appdef.FieldName("field3"),
						},
					},
					cmdName: {
						appdef.OperationKind_Execute: nil,
					},
					queryName: {
						appdef.OperationKind_Execute: nil,
					},
				},
			},
			// admin test
			{
				role: admin,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					viewName: {
						appdef.OperationKind_Select: nil,
					},
					cDocName: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
					oDocName: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
					queryName: {
						appdef.OperationKind_Execute: nil,
					},
					cmdName: {
						appdef.OperationKind_Execute: nil,
					},
				},
			},
			// intruder test
			{
				role: intruder,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					viewName: {
						appdef.OperationKind_Select: nil,
					},
				},
			},
			// unknown test
			{
				role: appdef.NewQName("test", "unknown"),
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{},
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.role.String(), func(t *testing.T) {
				got := map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{}
				for typ, ops := range acl.PublishedTypes(ws, tt.role) {
					if typ.QName().Pkg() == appdef.SysPackage {
						continue // skip system types
					}
					got[typ.QName()] = maps.Collect(ops)
				}
				require.Equal(tt.want, got)
			})
		}
	})

	t.Run("PublishedTypes should be breakable", func(t *testing.T) {
		typesCnt := 0
		for _, ops := range acl.PublishedTypes(app.Workspace(wsName), admin) {
			typesCnt++
			opsCnt := 0
			for range ops {
				opsCnt++
				break
			}
			require.Equal(1, opsCnt)
			break
		}
		require.Equal(1, typesCnt)
	})
}

func TestPublishedTypesWSInheritances(t *testing.T) {
	// # Test plan:
	// ┌─────────────────────────┬─────────────┐
	// │       workspaces        │  resources  │
	// ├───────────┬─────────────┼──────┬──────┤
	// │ ancestor  │ descendants │ doc1 │ doc2 │
	// ├───────────┴─────────────┼──────┼──────┤
	// │ abstractWS              │ S--  │ S--  │ grant select all tables to role
	// │     ▲     ┌─◄─ ws1      │ SIU  │ S--  │ grant all on doc1 to role
	// │     └─————┼─◄─ ws2      │ S--  │ SIU  │ grant all on doc2 to r2, grant r2 to role (#3127)
	// │           └─◄─ ws3      │ ---  │ ---  │ revoke all on doc1, doc2 from role
	// └─────────────────────────┴──────┴──────┘

	require := require.New(t)

	var app appdef.IAppDef

	awsName := appdef.NewQName("test", "abstractWS")
	ws1Name := appdef.NewQName("test", "ws1")
	ws2Name := appdef.NewQName("test", "ws2")
	ws3Name := appdef.NewQName("test", "ws3")

	roleName := appdef.NewQName("test", "role")
	r2Name := appdef.NewQName("test", "r2")

	doc1Name := appdef.NewQName("test", "doc1")
	doc2Name := appdef.NewQName("test", "doc2")

	t.Run("should be ok to build application", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		aws := adb.AddWorkspace(awsName)
		_ = aws.AddCDoc(doc1Name)
		_ = aws.AddCDoc(doc2Name)
		_ = aws.AddRole(roleName)
		aws.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, filter.AllWSTables(awsName), nil, roleName, "grant select all tables to role")

		ws1 := adb.AddWorkspace(ws1Name)
		ws1.SetAncestors(awsName)
		ws1.GrantAll(filter.QNames(doc1Name), roleName, "grant all on doc1 to role")

		ws2 := adb.AddWorkspace(ws2Name)
		ws2.SetAncestors(awsName)
		_ = ws2.AddRole(r2Name)
		ws2.GrantAll(filter.QNames(doc2Name), r2Name, "grant all on doc2 to r2")
		ws2.GrantAll(filter.WSTypes(ws2Name, appdef.TypeKind_Role), roleName, "grant {ALL WS ROLES} to role") // #3127

		ws3 := adb.AddWorkspace(ws3Name)
		ws3.SetAncestors(awsName)
		ws3.RevokeAll(filter.QNames(doc1Name, doc2Name), roleName, "revoke all on doc1, doc2 from role")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	require.NotNil(app)

	t.Run("test PublishedTypes", func(t *testing.T) {
		var tests = []struct {
			ws   appdef.QName
			want map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName
		}{
			{
				ws: awsName,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					doc1Name: {
						appdef.OperationKind_Select: nil,
					},
					doc2Name: {
						appdef.OperationKind_Select: nil,
					},
				},
			},
			{
				ws: ws1Name,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					doc1Name: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
					doc2Name: {
						appdef.OperationKind_Select: nil,
					},
				},
			},
			{
				ws: ws2Name,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{
					doc1Name: {
						appdef.OperationKind_Select: nil,
					},
					doc2Name: {
						appdef.OperationKind_Select:     nil,
						appdef.OperationKind_Insert:     nil,
						appdef.OperationKind_Update:     nil,
						appdef.OperationKind_Activate:   nil,
						appdef.OperationKind_Deactivate: nil,
					},
				},
			},
			{
				ws:   ws3Name,
				want: map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.ws.Entity(), func(t *testing.T) {
				ws := app.Workspace(tt.ws)
				got := map[appdef.QName]map[appdef.OperationKind]*[]appdef.FieldName{}
				for typ, ops := range acl.PublishedTypes(ws, roleName) {
					if typ.QName().Pkg() == appdef.SysPackage {
						continue // skip system types
					}
					got[typ.QName()] = maps.Collect(ops)
				}
				require.Equal(tt.want, got)
			})
		}
	})
}
