/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_IsOperationAllowed(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	docName := appdef.NewQName("test", "doc")
	queryName := appdef.NewQName("test", "qry")
	cmdName := appdef.NewQName("test", "cmd")

	reader := appdef.NewQName("test", "reader")
	writer := appdef.NewQName("test", "writer")
	intruder := appdef.NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := wsb.AddCDoc(docName)
		doc.
			AddField("field1", appdef.DataKind_int32, true).
			AddField("hiddenField", appdef.DataKind_int32, false).
			AddField("field3", appdef.DataKind_int32, false)

		qry := wsb.AddQuery(queryName)
		qry.SetResult(docName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetParam(appdef.QNameANY)

		_ = wsb.AddRole(reader)
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName}, nil, reader, "grant select doc.* to reader")
		wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName}, []appdef.FieldName{"hiddenField"}, reader, "revoke select doc.field1 from reader")
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{queryName}, nil, reader, "grant execute query to reader")

		_ = wsb.AddRole(writer)
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Insert}, []appdef.QName{docName}, nil, writer, "grant insert doc.* to writer")
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Update}, []appdef.QName{docName}, []appdef.FieldName{"field1", "hiddenField", "field3"}, writer, "grant update doc.field[1,2,3] to writer")
		wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Update}, []appdef.QName{docName}, []appdef.FieldName{"hiddenField"}, writer, "revoke update doc.hiddenField from writer")
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName}, nil, writer, "grant execute cmd to writer")

		_ = wsb.AddRole(intruder)
		wsb.RevokeAll([]appdef.QName{docName}, intruder, "revoke all access to doc from intruder")
		wsb.RevokeAll([]appdef.QName{queryName, cmdName}, intruder, "revoke all access to functions from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	selectableDocFields := []appdef.FieldName{
		appdef.SystemField_QName, appdef.SystemField_ID, appdef.SystemField_IsActive,
		"field1", "field3"}
	t.Run("test IsAllowed", func(t *testing.T) {
		var tests = []struct {
			name          string
			op            appdef.OperationKind
			res           appdef.QName
			fields        []appdef.FieldName
			role          appdef.QName
			allowed       bool
			allowedFields []appdef.FieldName
		}{
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
				allowedFields: nil,
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
				name:          "deny execute query for writer",
				op:            appdef.OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          writer,
				allowed:       false,
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := IsOperationAllowed(app, tt.op, tt.res, tt.fields, []appdef.QName{tt.role})
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})

	t.Run("test IsAllowed with multiple roles", func(t *testing.T) {
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := IsOperationAllowed(app, tt.op, tt.res, tt.fields, tt.role)
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})

	t.Run("test IsAllowed with errors", func(t *testing.T) {
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
			{
				name:   "role not found",
				op:     appdef.OperationKind_Execute,
				res:    cmdName,
				role:   []appdef.QName{appdef.NewQName("test", "unknown")},
				error:  appdef.ErrNotFoundError,
				errHas: "test.unknown",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := IsOperationAllowed(app, tt.op, tt.res, tt.fields, tt.role)
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
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		_ = wsb.AddRole(reader)
		_ = wsb.AddRole(writer)

		wsb.AddRole(worker).Grant(
			[]appdef.OperationKind{appdef.OperationKind_Inherits},
			[]appdef.QName{reader, writer}, nil, "grant reader and writer roles to worker")

		wsb.AddRole(owner).GrantAll(
			[]appdef.QName{worker}, "grant worker role to owner")

		wsb.AddRole(admin).GrantAll(
			[]appdef.QName{owner}, "grant owner role to admin")

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
				roles := RecursiveRoleAncestors(appdef.Role(app.Type, tt.role))
				require.ElementsMatch(tt.result, roles)
			})
		}
	})
}
