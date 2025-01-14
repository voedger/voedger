/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts/internal/acl"
	"github.com/voedger/voedger/pkg/goutils/set"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_IsOperationAllowed(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")
	queryName := appdef.NewQName("test", "qry")
	cmdName := appdef.NewQName("test", "cmd")
	tagName := appdef.NewQName("test", "tag")

	reader := appdef.NewQName("test", "reader")
	writer := appdef.NewQName("test", "writer")
	intruder := appdef.NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagName, "test tag")

		doc := wsb.AddCDoc(docName)
		doc.
			AddField("field1", appdef.DataKind_int32, true).
			AddField("hiddenField", appdef.DataKind_int32, false).
			AddField("field3", appdef.DataKind_int32, false)
		doc.SetTag(tagName)

		qry := wsb.AddQuery(queryName)
		qry.SetResult(docName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetParam(appdef.QNameANY)

		_ = wsb.AddRole(reader)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			reader,
			"grant select any CDoc with tag to reader")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName),
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
			[]appdef.OperationKind{appdef.OperationKind_Insert},
			filter.And(filter.WSTypes(wsName, appdef.TypeKind_CDoc), filter.Tags(tagName)),
			nil,
			writer,
			"grant insert any CDoc with tag to writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(docName),
			[]appdef.FieldName{"field1", "hiddenField", "field3"},
			writer,
			"grant update doc.field[1,2,3] to writer")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(docName),
			[]appdef.FieldName{"hiddenField"},
			writer,
			"revoke update doc.hiddenField from writer")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.AllWSFunctions(wsName),
			nil,
			writer,
			"grant execute all commands and queries to writer")

		_ = wsb.AddRole(intruder)
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

	selectableDocFields := []appdef.FieldName{
		appdef.SystemField_QName, appdef.SystemField_ID, appdef.SystemField_IsActive,
		"field1", "field3"}
	allDocFields := []appdef.FieldName{
		appdef.SystemField_QName, appdef.SystemField_ID, appdef.SystemField_IsActive,
		"field1", "hiddenField", "field3"}
	t.Run("test IsOperationAllowed", func(t *testing.T) {
		var tests = []struct {
			name          string
			op            appdef.OperationKind
			res           appdef.QName
			fields        []appdef.FieldName
			role          appdef.QName
			allowed       bool
			allowedFields []appdef.FieldName
		}{
			// iauthnz.QNameRoleSystem test
			{
				name:          "allow select * from doc for system",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          appdef.QNameRoleSystem,
				allowed:       true,
				allowedFields: allDocFields,
			},
			{
				name:          "allow insert to doc for system",
				op:            appdef.OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          appdef.QNameRoleSystem,
				allowed:       true,
				allowedFields: allDocFields,
			},
			{
				name:          "allow update doc for system",
				op:            appdef.OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          appdef.QNameRoleSystem,
				allowed:       true,
				allowedFields: allDocFields,
			},
			// reader tests
			{
				name:          "allow select doc.field1 for reader",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        []appdef.FieldName{"field1"},
				role:          reader,
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select doc.hiddenField for reader",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        []appdef.FieldName{"hiddenField"},
				role:          reader,
				allowed:       false,
				allowedFields: selectableDocFields,
			},
			{
				name:          "allow select ? from doc for reader",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select * from doc for reader",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        []appdef.FieldName{"field1", "hiddenField", "field3"},
				role:          reader,
				allowed:       false,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny insert doc for reader",
				op:            appdef.OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny update doc for reader",
				op:            appdef.OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow execute query for reader",
				op:            appdef.OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          reader,
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for reader",
				op:            appdef.OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			// writer tests
			{
				name:          "deny select ? from doc for writer",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow insert to doc for writer",
				op:            appdef.OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: allDocFields,
			},
			{
				name:          "allow update doc for writer",
				op:            appdef.OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: []appdef.FieldName{"field1", "field3"},
			},
			{
				name:          "deny update doc.hiddenField for writer",
				op:            appdef.OperationKind_Update,
				res:           docName,
				fields:        []appdef.FieldName{"hiddenField"},
				role:          writer,
				allowed:       false,
				allowedFields: []appdef.FieldName{"field1", "field3"},
			},
			{
				name:          "allow execute cmd for writer",
				op:            appdef.OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "allow execute query for writer",
				op:            appdef.OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: nil,
			},
			// intruder tests
			{
				name:          "deny select ? from doc for intruder",
				op:            appdef.OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny insert doc for intruder",
				op:            appdef.OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny update doc for intruder",
				op:            appdef.OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny execute query for intruder",
				op:            appdef.OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for intruder",
				op:            appdef.OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, []appdef.QName{tt.role})
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})

	t.Run("test IsOperationAllowed with multiple roles", func(t *testing.T) {
		var tests = []struct {
			name          string
			op            appdef.OperationKind
			res           appdef.QName
			fields        []appdef.FieldName
			role          []appdef.QName
			allowed       bool
			allowedFields []appdef.FieldName
		}{
			{
				name:          "allow select doc for [reader, writer]",
				op:            appdef.OperationKind_Select,
				res:           docName,
				role:          []appdef.QName{reader, writer},
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select doc for [reader, intruder]",
				op:            appdef.OperationKind_Select,
				res:           docName,
				role:          []appdef.QName{reader, intruder},
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow execute cmd for [reader, writer]",
				op:            appdef.OperationKind_Execute,
				res:           cmdName,
				role:          []appdef.QName{reader, writer},
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for [writer, intruder]",
				op:            appdef.OperationKind_Execute,
				res:           cmdName,
				role:          []appdef.QName{writer, intruder},
				allowed:       false,
				allowedFields: nil,
			},
		}
		ws := app.Workspace(wsName)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, tt.role)
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
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
			errHas string
		}{
			{
				name:   "unsupported operation",
				op:     appdef.OperationKind_Inherits,
				res:    docName,
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
				name:   "structure not found",
				op:     appdef.OperationKind_Select,
				res:    cmdName,
				role:   []appdef.QName{reader},
				error:  appdef.ErrNotFoundError,
				errHas: cmdName.String(),
			},
			{
				name:   "function not found",
				op:     appdef.OperationKind_Execute,
				res:    docName,
				role:   []appdef.QName{writer},
				error:  appdef.ErrNotFoundError,
				errHas: docName.String(),
			},
			{
				name:   "field not found",
				op:     appdef.OperationKind_Update,
				res:    docName,
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
				allowed, allowedFields, err := acl.IsOperationAllowed(ws, tt.op, tt.res, tt.fields, tt.role)
				require.Error(err, require.Is(tt.error), require.Has(tt.errHas))
				require.False(allowed)
				require.Nil(allowedFields)
			})
		}
	})
}

func TestRecursiveRoleAncestors(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	reader := appdef.NewQName("test", "reader")
	writer := appdef.NewQName("test", "writer")
	worker := appdef.NewQName("test", "worker")
	owner := appdef.NewQName("test", "owner")
	admin := appdef.NewQName("test", "adm")

	t.Run("should be ok to build application with roles", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		_ = wsb.AddRole(reader)
		_ = wsb.AddRole(writer)

		wsb.AddRole(worker).Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			filter.QNames(reader, writer), nil, "grant reader and writer roles to worker")

		wsb.AddRole(owner).GrantAll(
			filter.QNames(worker), "grant worker role to owner")

		wsb.AddRole(admin).GrantAll(
			filter.QNames(owner), "grant owner role to admin")

		app = adb.MustBuild()
	})

	t.Run("test RecursiveRoleAncestors", func(t *testing.T) {
		var tests = []struct {
			role   appdef.QName
			result []appdef.QName
		}{
			{reader, []appdef.QName{reader}},
			{writer, []appdef.QName{writer}},
			{worker, []appdef.QName{worker, reader, writer}},
			{owner, []appdef.QName{owner, worker, reader, writer}},
			{admin, []appdef.QName{admin, owner, worker, reader, writer}},
		}
		for _, tt := range tests {
			t.Run(tt.role.String(), func(t *testing.T) {
				roles := acl.RecursiveRoleAncestors(appdef.Role(app.Type, tt.role))
				require.ElementsMatch(tt.result, roles)
			})
		}
	})
}

func TestWSInheritances(t *testing.T) {
	// #3113: IsOperationAllowed should use request workspace
	//
	// # Test plan:
	// ┌─────────────────────────┬─────────────┐
	// │       workspaces        │  resources  │
	// ├───────────┬─────────────┼──────┬──────┤
	// │ ancestor  │ descendants │ doc1 │ doc2 │
	// ├───────────┴─────────────┼──────┼──────┤
	// │ abstractWS              │ S--  │ S--  │ grant select all tables to role
	// │     ▲     ┌─◄─ ws1      │ SIU  │ S--  │ grant all on doc1 to role
	// │     └─————┼─◄─ ws2      │ S--  │ SIU  │ grant all on doc2 to role
	// │           └─◄─ ws3      │ ---  │ ---  │ revoke all on doc1, doc2 from role
	// └─────────────────────────┴──────┴──────┘

	require := require.New(t)

	var app appdef.IAppDef

	awsName := appdef.NewQName("test", "abstractWS")
	ws1Name := appdef.NewQName("test", "ws1")
	ws2Name := appdef.NewQName("test", "ws2")
	ws3Name := appdef.NewQName("test", "ws3")

	roleName := appdef.NewQName("test", "role")

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
		ws2.GrantAll(filter.QNames(doc2Name), roleName, "grant all on doc2 to role")

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
						got, _, err := acl.IsOperationAllowed(ws, op, doc, nil, []appdef.QName{roleName})
						require.NoError(err)
						require.Equal(want, got, "%s %s", doc, op)
					}
				}
			})
		}
	})
}
