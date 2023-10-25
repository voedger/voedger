/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"gopkg.in/yaml.v2"
)

func loadExportedPackage(folderPath string, qn string) (*PackageSchemaAST, error) {

	var pathToFolder string

	pathToFolder = filepath.Join(folderPath, ExportedPkgFolder)

	path := strings.Split(qn, "/")
	if len(path) == 0 {
		return nil, ErrNoQualifiedName
	}

	for i, name := range path {
		if i == len(path)-1 {
			continue
		}
		pathToFolder = filepath.Join(pathToFolder, name)
	}

	return ParsePackageDir(qn, &PathReader{pathToFolder}, path[len(path)-1])
}

func loadExportedAppsInfo(folderPath string) (*ExportedAppsInfo, error) {
	bytes, err := os.ReadFile(filepath.Join(folderPath, ExportedAppsFile))
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

func loadExportedApps(folderPath string) ([]ExportedApp, error) {

	exportedApps := make([]ExportedApp, 0)

	expInfo, err := loadExportedAppsInfo(folderPath)
	if err != nil {
		return nil, err
	}
	for _, eapp := range expInfo.Apps {
		packages := make([]*PackageSchemaAST, 0)
		appPackage, err := loadExportedPackage(folderPath, eapp.Package)
		if err != nil {
			return nil, err
		}
		packages = append(packages, appPackage)
		appStmt, err := findApplication(appPackage)
		if err != nil {
			return nil, err
		}
		for _, use := range appStmt.Uses {
			pkgQN := getQualifiedPackageName(use.Name, appPackage.Ast)
			pkg, err := loadExportedPackage(folderPath, pkgQN)
			if err != nil {
				return nil, err
			}
			packages = append(packages, pkg)
		}

		pkgSys, err := loadExportedPackage(folderPath, appdef.SysPackage)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkgSys)

		appSchema, err := BuildAppSchema(packages)
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
