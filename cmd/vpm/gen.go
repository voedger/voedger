/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/parser"
)

//go:embed templates/*
var fsTemplates embed.FS

func newGenCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "generate",
	}
	cmd.AddCommand(newGenOrmCmd())

	initGlobalFlags(cmd, &params)
	return cmd
}

func newGenOrmCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "orm [--header-file]",
		Short: "generate ORM",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			compileRes, err := compile(params.WorkingDir)
			if err != nil {
				return err
			}
			return genOrm(compileRes, params)
		},
	}
	cmd.Flags().StringVarP(&params.HeaderFile, "header-file", "", "", " path to file to insert as a header to generated files")
	return cmd
}

// genOrm generates ORM from the given working directory
func genOrm(compileRes *compileResult, params vpmParams) error {
	ormDirPath, err := createOrmDir(params.TargetDir)
	if err != nil {
		return err
	}

	appDef, err := appDefFromCompiled(compileRes)
	if err != nil {
		return err
	}

	headerContent, err := getHeaderFileContent(params.HeaderFile)
	if err != nil {
		return err
	}

	for qpn, packageAst := range compileRes.appSchemaAST.Packages {
		packageData, err := fillOrmPackageData(qpn, packageAst, appDef, headerContent)
		if err != nil {
			return err
		}

		ormFile, err := generateOrmFile(packageData)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(ormDirPath, fmt.Sprintf("package_%s.go", qpn)), ormFile, defaultPermissions); err != nil {
			return err
		}
	}

	return nil
}

func fillOrmPackageData(qpn string, packageAst *parser.PackageSchemaAST, appDef appdef.IAppDef, headerFileContent string) (ormPackageData, error) {
	pkgData := ormPackageData{
		Name:              qpn,
		HeaderFileContent: headerFileContent,
		Imports:           []string{"import exttinygo \"github.com/voedger/exttinygo\""},
		Items:             make([]ormTableData, 0),
	}

	var qNames []appdef.QName
	appDef.DataTypes(false, func(data appdef.IData) {
		qNames = append(qNames, data.QName())
	})

	for _, stmt := range packageAst.Ast.Statements {
		if stmt.Table != nil {
			tableData := ormTableData{
				Package:    pkgData,
				Name:       stmt.Table.GetName(),
				Type:       fmt.Sprintf("%sTable", stmt.Table.Name),
				SqlContent: "Here will be SQL content of the table.",
				Fields:     make([]ormFieldData, 0),
			}

			for _, field := range stmt.Table.Items {
				fieldData := ormFieldData{
					Type: appDef.Data(appdef.NewQName(qpn, stmt.Table.GetName())),
					//Name:          field.GetName(),
					GetMethodName: fmt.Sprintf("Get%s", "Field"),
				}

				tableData.Fields = append(tableData.Fields, fieldData)
			}

			pkgData.Items = append(pkgData.Items, tableData)
		}
	}
	return pkgData, nil
}

func generateOrmFile(templateData interface{}) ([]byte, error) {
	switch t := templateData.(type) {
	case ormPackageData:
		return fillTemplate("solid_package", t)
	default:
		return nil, fmt.Errorf("unknown template data type: %T", t)
	}
}

func fillTemplate(templateName string, payload interface{}) ([]byte, error) {
	templateContent, err := fsTemplates.ReadFile(fmt.Sprintf("templates/%s.txt", templateName))
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	tmpl := template.New(templateName)
	tmpl, err = tmpl.Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var filledTemplate bytes.Buffer
	if err := tmpl.Execute(&filledTemplate, payload); err != nil {
		return nil, fmt.Errorf("failed to fill template: %w", err)
	}

	return format.Source(filledTemplate.Bytes())
}

func getHeaderFileContent(headerFilePath string) (string, error) {
	if headerFilePath == "" {
		return "", nil
	}

	headerFileContent, err := os.ReadFile(headerFilePath)
	if err != nil {
		return "", err
	}

	return string(headerFileContent), nil
}

func createOrmDir(dir string) (string, error) {
	ormDirPath := filepath.Join(dir, ormDirName)
	if _, err := os.Stat(ormDirPath); os.IsExist(err) {
		if err := os.RemoveAll(ormDirPath); err != nil {
			return "", err
		}
	}
	return ormDirPath, os.MkdirAll(ormDirPath, defaultPermissions)
}
