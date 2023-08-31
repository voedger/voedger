/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"
	"runtime"

	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/parser"
)

func Parse(fsi embed.FS, schemaName string, ep extensionpoints.IExtensionPoint) {
	dirEntries, err := fsi.ReadDir(".")
	if err != nil {
		//notest
		panic(err)
	}
	for _, dirEntry := range dirEntries {
		sqlContent, err := fsi.ReadFile(dirEntry.Name())
		if err != nil {
			// notest
			panic(err)
		}
		fileSchemaAST, err := parser.ParseFile(dirEntry.Name(), string(sqlContent))
		if err != nil {
			// notest
			_, file, _, _ := runtime.Caller(1)
			panic("from " + file + ": " + err.Error())
		}
		epFileSchemaASTs := ep.ExtensionPoint(EPPackageSchemasASTs)
		epSysFileSchemaASTs := epFileSchemaASTs.ExtensionPoint(schemaName)
		epSysFileSchemaASTs.Add(fileSchemaAST)
	}
}
