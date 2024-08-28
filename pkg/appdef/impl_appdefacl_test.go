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
			AddField("field2", DataKind_int32, false).
			AddField("field3", DataKind_int32, false)

		qry := adb.AddQuery(queryName)
		qry.SetResult(docName)

		cmd := adb.AddCommand(cmdName)
		cmd.SetParam(QNameANY)

		_ = adb.AddRole(reader)
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, nil, reader, "grant select doc.* to reader")
		adb.Revoke([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"field2"}, reader, "revoke select doc.field1 from reader")
		adb.Grant([]OperationKind{OperationKind_Execute}, []QName{queryName}, nil, reader, "grant execute query to reader")

		_ = adb.AddRole(writer)
		adb.Grant([]OperationKind{OperationKind_Insert}, []QName{docName}, nil, writer, "grant insert doc.* to writer")
		adb.Grant([]OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"field1", "field2", "field3"}, writer, "grant update doc.field[1,2,3] to writer")
		adb.Revoke([]OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"field2"}, writer, "revoke update doc.field2 from writer")
		adb.Grant([]OperationKind{OperationKind_Execute}, []QName{cmdName}, nil, writer, "grant execute cmd to writer")

		_ = adb.AddRole(intruder)
		adb.RevokeAll([]QName{docName}, intruder, "revoke all access to doc from intruder")
		adb.RevokeAll([]QName{queryName}, intruder, "revoke all access to query from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("test IsAllowed", func(t *testing.T) {
		allowedDocFields := []FieldName{SystemField_QName, SystemField_ID, SystemField_IsActive, "field1", "field3"}
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
				allowedFields: allowedDocFields,
			},
			{
				name:          "deny select doc.field2 for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        []FieldName{"field2"},
				role:          reader,
				allowed:       false,
				allowedFields: allowedDocFields,
			},
			{
				name:          "allow select ? from doc for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        nil,
				role:          reader,
				allowed:       true,
				allowedFields: allowedDocFields,
			},
			{
				name:          "deny select * from doc for reader",
				op:            OperationKind_Select,
				res:           docName,
				fields:        []FieldName{"field1", "field2", "field3"},
				role:          reader,
				allowed:       false,
				allowedFields: allowedDocFields,
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
				name:          "deny update doc.field2 for writer",
				op:            OperationKind_Update,
				res:           docName,
				fields:        []FieldName{"field2"},
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
				allowed, allowedFields := app.IsOperationAllowed(tt.op, tt.res, tt.fields, []QName{tt.role})
				require.Equal(tt.allowed, allowed)
				require.EqualValues(tt.allowedFields, allowedFields)
			})
		}
	})
}
