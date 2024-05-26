/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	query := `select * from test1.app1.123.sys.Table.456 where x = 1`
	dml, err := ParseQuery(query)
	require.NoError(err)
	expectedDML := DML{
		AppQName: istructs.NewAppQName("test1", "app1"),
		QName:    appdef.NewQName("sys", "Table"),
		Kind:     DMLKind_Select,
		Location: Location{
			ID:   123,
			Kind: LocationKind_WSID,
		},
		EntityID: 456,
		CleanSQL: "select * from sys.Table where x = 1",
	}
	require.Equal(expectedDML, dml)
}

func TestCases(t *testing.T) {
	require := require.New(t)
	test1App1 := istructs.NewAppQName("test1", "app1")
	sysTable := appdef.NewQName("sys", "Table")
	cases := []struct {
		query string
		dml   DML
	}{
		{
			"select * from sys.Table where x = 1",
			DML{
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
			},
		},
		{
			"select * from test1.app1.sys.Table where x = 1",
			DML{
				AppQName: test1App1,
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
			},
		},
		{
			"select * from test1.app1.123.sys.Table where x = 1",
			DML{
				AppQName: test1App1,
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
				Location: Location{
					ID:   123,
					Kind: LocationKind_WSID,
				},
			},
		},
		{
			"select * from test1.app1.123.sys.Table.456 where x = 1",
			DML{
				AppQName: test1App1,
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
				Location: Location{
					ID:   123,
					Kind: LocationKind_WSID,
				},
				EntityID: 456,
			},
		},
		{
			"select * from test1.app1.a123.sys.Table.456 where x = 1",
			DML{
				AppQName: test1App1,
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
				Location: Location{
					ID:   123,
					Kind: LocationKind_AppWSNum,
				},
				EntityID: 456,
			},
		},
		{
			`select * from test1.app1."login".sys.Table.456 where x = 1`,
			DML{
				AppQName: test1App1,
				Kind:     DMLKind_Select,
				QName:    sysTable,
				CleanSQL: "select * from sys.Table where x = 1",
				Location: Location{
					ID:   140737488407312,
					Kind: LocationKind_PseudoWSID,
				},
				EntityID: 456,
			},
		},
	}
	for _, c := range cases {
		dml, err := ParseQuery(c.query)
		require.NoError(err)
		require.Equal(c.dml, dml)
	}
}
