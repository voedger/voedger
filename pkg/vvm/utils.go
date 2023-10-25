/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package vvm

import (
	"embed"
	"path"
	"path/filepath"

	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/parser"
)

func readFileSchemaAST(dir string, fsi embed.FS) (fileSchemasAST []*parser.FileSchemaAST, err error) {
	dirEntries, err := fsi.ReadDir(".")
	if err != nil {
		return nil, err
	}
	for _, dirEntry := range dirEntries {
		fileName := dirEntry.Name()
		sqlContent, err := fsi.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		fullFilePath := path.Join(dir, fileName)

		fileSchemaAST, err := parser.ParseFile(fullFilePath, string(sqlContent))
		if err != nil {
			return nil, err
		}
		fileSchemasAST = append(fileSchemasAST, fileSchemaAST)
	}
	return
}

func ReadPackageSchemaAST(ep extensionpoints.IExtensionPoint) (packageSchemaASTs []*parser.PackageSchemaAST, err error) {
	epSqlFiles := ep.ExtensionPoint(apps.EPSchemasFS)
	epSqlFiles.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
		filesSchemasASTs := make([]*parser.FileSchemaAST, 0)
		qualifiedPackageName, _ := eKey.(string)
		epPackageSql := value.(extensionpoints.IExtensionPoint)
		epPackageSql.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
			dirAndPackageName := eKey.(string)
			dir := filepath.Dir(dirAndPackageName)

			fsi, _ := value.(embed.FS)
			fileSchemaASTs, err := readFileSchemaAST(dir, fsi)
			if err != nil {
				panic(err)
			}
			filesSchemasASTs = append(filesSchemasASTs, fileSchemaASTs...)
		})
		packageSchemaAST, err := parser.BuildPackageSchema(qualifiedPackageName, filesSchemasASTs)
		if err != nil {
			panic(err)
		}
		packageSchemaASTs = append(packageSchemaASTs, packageSchemaAST)
	})
	return
}
