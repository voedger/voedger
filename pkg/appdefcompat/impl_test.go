/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/parser"
)

//go:embed sql_example_syspkg/old.sql
var oldFS embed.FS

//go:embed sql_example_syspkg/new.sql
var newFS embed.FS

func getSysPackageAST(file parser.IReadFS) *parser.PackageSchemaAST {
	pkgSys, err := parser.ParsePackageDir(appdef.SysPackage, file, "sql_example_syspkg")
	if err != nil {
		panic(err)
	}
	return pkgSys
}

func Test_Basic(t *testing.T) {
	oldPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		getSysPackageAST(oldFS),
	})
	require.NoError(t, err)

	newPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		getSysPackageAST(newFS),
	})
	require.NoError(t, err)

	oldBuilder := appdef.New()
	require.NoError(t, parser.BuildAppDefs(oldPackages, oldBuilder))

	newBuilder := appdef.New()
	require.NoError(t, parser.BuildAppDefs(newPackages, newBuilder))

	compatErrors := checkBackwardCompatibility(oldBuilder, newBuilder)
	// 4 errors expected: 1 - Mismatch, 1 - NodeRemoved, 2 - OrderChanged
	require.Greater(t, len(compatErrors.Errors), 4)
}
