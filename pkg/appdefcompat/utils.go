/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package appdefcompat

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/parser"
	"gopkg.in/yaml.v2"
)

func LoadExportedApps(folderPath string) ([]ExportedApp, error) {

	exportedApps := make([]ExportedApp, 0)

	expInfo, err := loadExportedAppsInfo(folderPath)
	if err != nil {
		return nil, err
	}
	for _, eapp := range expInfo.Apps {
		packages := make([]*parser.PackageSchemaAST, 0)
		appPackage, err := loadExportedPackage(folderPath, eapp.Package)
		if err != nil {
			return nil, err
		}
		packages = append(packages, appPackage)
		appStmt, err := parser.FindApplication(appPackage)
		if err != nil {
			return nil, err
		}
		if appStmt != nil {
			for _, use := range appStmt.Uses {
				pkgQN := parser.GetQualifiedPackageName(use.Name, appPackage.Ast)
				pkg, err := loadExportedPackage(folderPath, pkgQN)
				if err != nil {
					return nil, err
				}
				packages = append(packages, pkg)
			}
		}

		pkgSys, err := loadExportedPackage(folderPath, appdef.SysPackage)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkgSys)

		appSchema, err := parser.BuildAppSchema(packages)
		if err != nil {
			return nil, err
		}
		exportedApps = append(exportedApps, ExportedApp{
			Ast:    appSchema,
			Ignore: append(expInfo.Ignore, eapp.Ignore...),
		})

	}

	return exportedApps, nil
}

func loadExportedAppsInfo(folderPath string) (*ExportedAppsInfo, error) {
	bytes, err := os.ReadFile(filepath.Join(folderPath, parser.ExportedAppsFile))
	if err != nil {
		return nil, err
	}
	var expInfo ExportedAppsInfo
	err = yaml.Unmarshal(bytes, &expInfo)
	if err != nil {
		return nil, err
	}
	return &expInfo, nil
}

func loadExportedPackage(folderPath string, qn string) (*parser.PackageSchemaAST, error) {

	var pathToFolder string

	pathToFolder = filepath.Join(folderPath, parser.ExportedPkgFolder)

	path := strings.Split(qn, "/")
	if len(path) == 0 {
		return nil, parser.ErrNoQualifiedName
	}

	for i, name := range path {
		if i == len(path)-1 {
			continue
		}
		pathToFolder = filepath.Join(pathToFolder, name)
	}

	return parser.ParsePackageDir(qn, &PathReader{pathToFolder}, path[len(path)-1])
}
