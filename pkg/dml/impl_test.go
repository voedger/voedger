/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package dml

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	query := `select * from test1.app1.123.sys.Table.456 where x = 1`
	op, err := ParseQuery(query)
	require.NoError(err)
	expectedDML := Op{
		AppQName: appdef.NewAppQName("test1", "app1"),
		QName:    appdef.NewQName("sys", "Table"),
		Kind:     OpKind_Select,
		Workspace: Workspace{
			ID:   123,
			Kind: WorkspaceKind_WSID,
		},
		EntityID:              456,
		CleanSQL:              "select * from sys.Table where x = 1",
		VSQLWithoutAppAndWSID: "select * from sys.Table.456 where x = 1",
	}
	require.Equal(expectedDML, op)
}

func TestCases(t *testing.T) {
	require := require.New(t)
	test1App1 := appdef.NewAppQName("test1", "app1")
	sysTable := appdef.NewQName("sys", "Table")
	cases := []struct {
		query string
		dml   Op
	}{
		{
			"select * from sys.Table where x = 1",
			Op{
				Kind:                  OpKind_Select,
				QName:                 sysTable,
				CleanSQL:              "select * from sys.Table where x = 1",
				VSQLWithoutAppAndWSID: "select * from sys.Table where x = 1",
			},
		},
		{
			"select * from test1.app1.sys.Table where x = 1",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_Select,
				QName:                 sysTable,
				CleanSQL:              "select * from sys.Table where x = 1",
				VSQLWithoutAppAndWSID: "select * from sys.Table where x = 1",
			},
		},
		{
			"select * from test1.app1.123.sys.Table where x = 1",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_Select,
				QName:                 sysTable,
				CleanSQL:              "select * from sys.Table where x = 1",
				VSQLWithoutAppAndWSID: "select * from sys.Table where x = 1",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_WSID,
				},
			},
		},
		{
			"select * from test1.app1.123.sys.Table.456 where x = 1",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_Select,
				QName:                 sysTable,
				CleanSQL:              "select * from sys.Table where x = 1",
				VSQLWithoutAppAndWSID: "select * from sys.Table.456 where x = 1",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_WSID,
				},
				EntityID: 456,
			},
		},
		{
			"select * from test1.app1.a123.sys.Table.456 where x = 1",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_Select,
				QName:                 sysTable,
				CleanSQL:              "select * from sys.Table where x = 1",
				VSQLWithoutAppAndWSID: "select * from sys.Table.456 where x = 1",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
				EntityID: 456,
			},
		},
		{
			`select * from te-st_1.ap-p1_."login".sy_-s.Ta-b_le.456 where x = 1`,
			Op{
				AppQName:              appdef.NewAppQName("te-st_1", "ap-p1_"),
				Kind:                  OpKind_Select,
				QName:                 appdef.NewQName("sy_-s", "Ta-b_le"),
				CleanSQL:              "select * from sy_-s.Ta-b_le where x = 1",
				VSQLWithoutAppAndWSID: "select * from sy_-s.Ta-b_le.456 where x = 1",
				Workspace: Workspace{
					ID:   140737488407312,
					Kind: WorkspaceKind_PseudoWSID,
				},
				EntityID: 456,
			},
		},
		{
			"update test1.app1.a123.sys.Table.456 set a = b where x = 1",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_UpdateTable,
				QName:                 sysTable,
				CleanSQL:              "update sys.Table set a = b where x = 1",
				VSQLWithoutAppAndWSID: "update sys.Table.456 set a = b where x = 1",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
				EntityID: 456,
			},
		},
		{
			"update corrupted test1.app1.a123.sys.Table.456",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_UpdateCorrupted,
				QName:                 sysTable,
				CleanSQL:              "",
				VSQLWithoutAppAndWSID: "update corrupted sys.Table.456",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
				EntityID: 456,
			},
		},
		{
			"unlogged update test1.app1.a123.sys.Table set a = b where x = y",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_UnloggedUpdate,
				QName:                 sysTable,
				CleanSQL:              "update sys.Table set a = b where x = y",
				VSQLWithoutAppAndWSID: "unlogged update sys.Table set a = b where x = y",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
			},
		},
		{
			"unlogged insert test1.app1.a123.sys.Table set a = b",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_UnloggedInsert,
				QName:                 sysTable,
				CleanSQL:              "update sys.Table set a = b",
				VSQLWithoutAppAndWSID: "unlogged insert sys.Table set a = b",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
			},
		},
		{
			"insert test1.app1.a123.sys.Table set a = b",
			Op{
				AppQName:              test1App1,
				Kind:                  OpKind_InsertTable,
				QName:                 sysTable,
				CleanSQL:              "update sys.Table set a = b",
				VSQLWithoutAppAndWSID: "insert sys.Table set a = b",
				Workspace: Workspace{
					ID:   123,
					Kind: WorkspaceKind_AppWSNum,
				},
			},
		},
	}
	for _, c := range cases {
		op, err := ParseQuery(c.query)
		require.NoError(err, c.query)
		require.Equal(c.dml, op, c.query)
	}
}

func TestErrors(t *testing.T) {
	cases := map[string]string{
		"":         "invalid query format",
		" ":        "invalid query format",
		"ddsddsds": "invalid query format",
		"unlogged update test1.app1.9999999999999999999999.sys.Table set a = b where x = y":   "value out of range",
		"unlogged update test1.app1.1.sys.Table.9999999999999999999999 set a = b where x = y": "value out of range",
		"unlogged update test1.app1.1.sys.Table.0 set a = b where x = y":                      "provided offset or ID must not be 0",
	}

	for query, expectedError := range cases {
		_, err := ParseQuery(query)
		require.ErrorContains(t, err, expectedError)
	}
}
