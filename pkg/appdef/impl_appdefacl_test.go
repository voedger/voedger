/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_IsOperationAllowed(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	docName := NewQName("test", "doc")
	queryName := NewQName("test", "qry")
	cmdName := NewQName("test", "cmd")

	reader := NewQName("test", "reader")
	writer := NewQName("test", "writer")
	intruder := NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddCDoc(docName)
		doc.
			AddField("field1", DataKind_int32, true).
			AddField("hiddenField", DataKind_int32, false).
			AddField("field3", DataKind_int32, false)

		qry := adb.AddQuery(queryName)
		qry.SetResult(docName)

		cmd := adb.AddCommand(cmdName)
		cmd.SetParam(QNameANY)

		_ = adb.AddRole(reader)
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, nil, reader, "grant select doc.* to reader")
		adb.Revoke([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"hiddenField"}, reader, "revoke select doc.field1 from reader")
		adb.Grant([]OperationKind{OperationKind_Execute}, []QName{queryName}, nil, reader, "grant execute query to reader")

		_ = adb.AddRole(writer)
		adb.Grant([]OperationKind{OperationKind_Insert}, []QName{docName}, nil, writer, "grant insert doc.* to writer")
		adb.Grant([]OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"field1", "hiddenField", "field3"}, writer, "grant update doc.field[1,2,3] to writer")
		adb.Revoke([]OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"hiddenField"}, writer, "revoke update doc.hiddenField from writer")
		adb.Grant([]OperationKind{OperationKind_Execute}, []QName{cmdName}, nil, writer, "grant execute cmd to writer")

		_ = adb.AddRole(intruder)
		adb.RevokeAll([]QName{docName}, intruder, "revoke all access to doc from intruder")
		adb.RevokeAll([]QName{queryName, cmdName}, intruder, "revoke all access to functions from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	selectableDocFields := []FieldName{SystemField_QName, SystemField_ID, SystemField_IsActive, "field1", "field3"}
	t.Run("test IsAllowed", func(t *testing.T) {
		var tests = []struct {
			name          string
			op            OperationKind
			res           QName
			fields        []FieldName
			role          QName
			allowed       bool
			allowedFields []FieldName
		}{
			// reader tests
			{
				name:          "allow select doc.field1 for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        []FieldName{"field1"},
				role:          reader,
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select doc.hiddenField for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        []FieldName{"hiddenField"},
				role:          reader,
				allowed:       false,
				allowedFields: selectableDocFields,
			},
			{
				name:          "allow select ? from doc for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select * from doc for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        []FieldName{"field1", "hiddenField", "field3"},
				role:          reader,
				allowed:       false,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny insert doc for reader",
				op:            OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny update doc for reader",
				op:            OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow execute query for reader",
				op:            OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          reader,
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for reader",
				op:            OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          reader,
				allowed:       false,
				allowedFields: nil,
			},
			// writer tests
			{
				name:          "deny select ? from doc for writer",
				op:            OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow insert to doc for writer",
				op:            OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "allow update doc for writer",
				op:            OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: []FieldName{"field1", "field3"},
			},
			{
				name:          "deny update doc.hiddenField for writer",
				op:            OperationKind_Update,
				res:           docName,
				fields:        []FieldName{"hiddenField"},
				role:          writer,
				allowed:       false,
				allowedFields: []FieldName{"field1", "field3"},
			},
			{
				name:          "allow execute cmd for writer",
				op:            OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          writer,
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "deny execute query for writer",
				op:            OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          writer,
				allowed:       false,
				allowedFields: nil,
			},
			// intruder tests
			{
				name:          "deny select ? from doc for intruder",
				op:            OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny insert doc for intruder",
				op:            OperationKind_Insert,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny update doc for intruder",
				op:            OperationKind_Update,
				res:           docName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny execute query for intruder",
				op:            OperationKind_Execute,
				res:           queryName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for intruder",
				op:            OperationKind_Execute,
				res:           cmdName,
				fields:        nil,
				role:          intruder,
				allowed:       false,
				allowedFields: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := app.IsOperationAllowed(tt.op, tt.res, tt.fields, []QName{tt.role})
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})

	t.Run("test IsAllowed with multiple roles", func(t *testing.T) {
		var tests = []struct {
			name          string
			op            OperationKind
			res           QName
			fields        []FieldName
			role          []QName
			allowed       bool
			allowedFields []FieldName
		}{
			{
				name:          "allow select doc for [reader, writer]",
				op:            OperationKind_Select,
				res:           docName,
				role:          []QName{reader, writer},
				allowed:       true,
				allowedFields: selectableDocFields,
			},
			{
				name:          "deny select doc for [reader, intruder]",
				op:            OperationKind_Select,
				res:           docName,
				role:          []QName{reader, intruder},
				allowed:       false,
				allowedFields: nil,
			},
			{
				name:          "allow execute cmd for [reader, writer]",
				op:            OperationKind_Execute,
				res:           cmdName,
				role:          []QName{reader, writer},
				allowed:       true,
				allowedFields: nil,
			},
			{
				name:          "deny execute cmd for [writer, intruder]",
				op:            OperationKind_Execute,
				res:           cmdName,
				role:          []QName{writer, intruder},
				allowed:       false,
				allowedFields: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := app.IsOperationAllowed(tt.op, tt.res, tt.fields, tt.role)
				require.NoError(err)
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})

	t.Run("test IsAllowed with errors", func(t *testing.T) {
		var tests = []struct {
			name   string
			op     OperationKind
			res    QName
			fields []FieldName
			role   []QName
			error  error
			errHas string
		}{
			{
				name:   "unsupported operation",
				op:     OperationKind_Inherits,
				res:    docName,
				role:   []QName{reader},
				error:  ErrUnsupportedError,
				errHas: "Inherits",
			},
			{
				name:   "resource not found",
				op:     OperationKind_Insert,
				res:    NewQName("test", "unknown"),
				role:   []QName{reader},
				error:  ErrNotFoundError,
				errHas: "test.unknown",
			},
			{
				name:   "structure not found",
				op:     OperationKind_Select,
				res:    cmdName,
				role:   []QName{reader},
				error:  ErrNotFoundError,
				errHas: cmdName.String(),
			},
			{
				name:   "function not found",
				op:     OperationKind_Execute,
				res:    docName,
				role:   []QName{writer},
				error:  ErrNotFoundError,
				errHas: docName.String(),
			},
			{
				name:   "field not found",
				op:     OperationKind_Update,
				res:    docName,
				fields: []FieldName{"unknown"},
				role:   []QName{writer},
				error:  ErrNotFoundError,
				errHas: "unknown",
			},
			{
				name:   "no participants",
				op:     OperationKind_Execute,
				res:    cmdName,
				error:  ErrMissedError,
				errHas: "participant",
			},
			{
				name:   "role not found",
				op:     OperationKind_Execute,
				res:    cmdName,
				role:   []QName{NewQName("test", "unknown")},
				error:  ErrNotFoundError,
				errHas: "test.unknown",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed, allowedFields, err := app.IsOperationAllowed(tt.op, tt.res, tt.fields, tt.role)
				require.Error(err, require.Is(tt.error), require.Has(tt.errHas))
				require.False(allowed)
				require.Nil(allowedFields)
			})
		}
	})
}
