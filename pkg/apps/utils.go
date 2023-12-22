/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
)

// includes sys
// all sql files should be in the root of the fs
func BuildAppDefFromFS(qualifiedPackageName string, fs parser.IReadFS, subDir string, additionalPackagesFS ...parser.PackageFS) (appdef.IAppDef, error) {
	appPackageAst, err := parser.ParsePackageDir(qualifiedPackageName, fs, subDir)
	if err != nil {
		return nil, err
	}
	sysPackageAST, err := parser.ParsePackageDir(appdef.SysPackage, sys.SysFS, ".")
	if err != nil {
		return nil, err
	}
	packagesAST := []*parser.PackageSchemaAST{appPackageAst, sysPackageAST}
	for _, adds := range additionalPackagesFS {
		additionalPackageAST, err := parser.ParsePackageDir(adds.QualifiedPackageName, adds.FS, ".")
		if err != nil {
			return nil, err
		}
		packagesAST = append(packagesAST, additionalPackageAST)
	}
	appSchema, err := parser.BuildAppSchema(packagesAST)
	if err != nil {
		return nil, err
	}

	adb := appdef.New()
	err = parser.BuildAppDefs(appSchema, adb)
	if err != nil {
		return nil, err
	}

	return adb.Build()
}
