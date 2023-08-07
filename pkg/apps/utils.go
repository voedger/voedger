/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"

	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/parser"
)

// package qualified name is the package the `fs` is declared at
func Parse(fs embed.FS, packageQualifedName string, ep extensionpoints.IExtensionPoint) {
	// legacy unTill table schemas
	dirEntries, err := embed.FS(fs).ReadDir(".")
	if err != nil {
		//notest
		panic(err)
	}
	for _, dirEntry := range dirEntries {
		sqlContent, err := embed.FS(fs).ReadFile(dirEntry.Name())
		if err != nil {
			// notest
			panic(err)
		}
		fileSchemaAST, err := parser.ParseFile(dirEntry.Name(), string(sqlContent))
		if err != nil {
			// notest
			panic(err)
		}
		epFileSchemaASTs := ep.ExtensionPoint(EPPackageSchemasASTs)
		epSysFileSchemaASTs := epFileSchemaASTs.ExtensionPoint(packageQualifedName)
		epSysFileSchemaASTs.Add(fileSchemaAST)
	}
}
