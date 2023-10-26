/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package vvm

import (
	"embed"
	"io/fs"
	"path"
	"path/filepath"

	"golang.org/x/exp/maps"

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

func readEmbeddedContent(dir, subDir string, fsi embed.FS) (contentMap map[string][]byte, err error) {
	subFS, err := fs.Sub(fsi, subDir)
	if err != nil {
		return
	}
	entries, err := fs.ReadDir(subFS, ".")
	if err != nil {
		return
	}
	contentMap = make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(subFS, entry.Name())
		if err != nil {
			return nil, err
		}
		fullFilePath := path.Join(dir, subDir, entry.Name())
		contentMap[fullFilePath] = content
	}
	return
}

func SchemaFilesContent(ep extensionpoints.IExtensionPoint, subDir string) (mapPackageContent apps.SchemasExportedContent, err error) {
	epSqlFiles := ep.ExtensionPoint(apps.EPSchemasFS)
	mapPackageContent = make(apps.SchemasExportedContent)
	epSqlFiles.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
		contentMaps := make(map[string][]byte)
		qualifiedPackageName, _ := eKey.(string)
		epPackageSql := value.(extensionpoints.IExtensionPoint)
		epPackageSql.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
			dirAndPackageName := eKey.(string)
			dir := filepath.Dir(dirAndPackageName)

			fsi, _ := value.(embed.FS)
			contentMap, err := readEmbeddedContent(dir, subDir, fsi)
			if err != nil {
				panic(err)
			}
			maps.Copy(contentMaps, contentMap)
		})
		mapPackageContent[qualifiedPackageName] = contentMaps
	})
	return
}
