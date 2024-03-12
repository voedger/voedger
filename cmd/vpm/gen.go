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
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
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
			compileRes, err := compile.Compile(params.Dir)
			if err != nil {
				return err
			}
			return genOrm(compileRes, params)
		},
	}
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	cmd.Flags().StringVarP(&params.HeaderFile, "header-file", "", "", " path to file to insert as a header to generated files")
	return cmd
}

// genOrm generates ORM from the given working directory
func genOrm(compileRes *compile.Result, params vpmParams) error {
	dir, err := createOrmDir(params.TargetDir)
	if err != nil {
		return err
	}

	headerContent, err := getHeaderFileContent(params.HeaderFile)
	if err != nil {
		return err
	}

	for pkgLocalName, objs := range gatherPackageObjs(compileRes.AppDef) {
		if len(objs) == 0 {
			continue
		}
		if err := generatePackage(pkgLocalName, objs, headerContent, dir); err != nil {
			return err
		}
	}
	return generateSysPackage(dir)
}

func gatherPackageObjs(appDef appdef.IAppDef) map[string][]interface{} {
	pkgObjs := make(map[string][]interface{}) // mapping of package local names to list of its objects
	pkgLocalNames := appDef.PackageLocalNames()
	for _, pkgLocalName := range pkgLocalNames {
		pkgObjs[pkgLocalName] = make([]interface{}, 0)
	}

	reg := regexp.MustCompile("(.*)\\.(.*)")

	appDef.Types(func(iType appdef.IType) {
		if workspace, ok := iType.(appdef.IWorkspace); ok {
			workspace.Types(func(iType appdef.IType) {
				qName := iType.QName().String()
				matches := reg.FindStringSubmatch(qName)
				pkgObjs[matches[1]] = append(pkgObjs[matches[1]], iType)
			})
		}
	})
	return pkgObjs
}

func generatePackage(pkgLocalName string, objs []interface{}, headerFileContent, dir string) error {
	pkgData, err := fillPackageData(pkgLocalName, objs, headerFileContent)
	if err != nil {
		return err
	}

	pkgFile, err := fillTemplate("package", pkgData)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, fmt.Sprintf("package_%s.go", pkgLocalName))
	return saveFile(filePath, pkgFile)
}

func generateSysPackage(dir string) error {
	sysFilePath := filepath.Join(dir, "sys.go")
	return saveFile(sysFilePath, []byte(sysContent))
}

func saveFile(filePath string, content []byte) error {
	return os.WriteFile(filePath, content, defaultPermissions)
}

func fillPackageData(pkgLocalName string, objs []interface{}, headerFileContent string) (ormPackageData, error) {
	pkgData := ormPackageData{
		Name:              pkgLocalName,
		HeaderFileContent: headerFileContent,
		Imports:           []string{"import exttinygo \"github.com/voedger/exttinygo\""},
		Items:             make([]ormTableData, 0),
	}

	for _, obj := range objs {
		switch t := obj.(type) {
		case appdef.IODoc:
			tableData := ormTableData{
				Package:    pkgData,
				TypeQName:  "typeQname",
				Name:       getName(t),
				Type:       getType(t),
				SqlContent: "Here will be SQL content of the table.",
				Fields:     make([]ormFieldData, 0),
			}

			for _, field := range t.Fields() {
				fieldType := getFieldType(field)
				if fieldType != unknownFieldType {
					fieldData := ormFieldData{
						Table:         tableData,
						Type:          getFieldType(field),
						Name:          field.Name(),
						GetMethodName: fmt.Sprintf("Get_%s", strings.ToLower(field.Name())),
					}
					tableData.Fields = append(tableData.Fields, fieldData)
				}
			}

			pkgData.Items = append(pkgData.Items, tableData)
		}
	}
	return pkgData, nil
}

func fillTemplate(templateName string, payload interface{}) ([]byte, error) {
	templateContent, err := fsTemplates.ReadFile(fmt.Sprintf("templates/%s.gotmpl", templateName))
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	t := template.New(templateName)
	t.Funcs(template.FuncMap{"capitalize": capitalizeFirst})
	t, err = t.Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var filledTemplate bytes.Buffer
	if err := t.Execute(&filledTemplate, payload); err != nil {
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

func getName(obj interface{}) string {
	return strings.ToLower(obj.(appdef.IType).QName().Entity())
}

func getType(obj interface{}) string {
	switch obj.(type) {
	case appdef.IODoc:
		return "ODoc"
	case appdef.ICDoc:
		return "CDoc"
	case appdef.IWDoc:
		return "WDoc"
	default:
		return unknownObjectType
	}
}

func getFieldType(field appdef.IField) string {
	switch field.DataKind() {
	case appdef.DataKind_bool:
		return "bool"
	case appdef.DataKind_int32:
		return "int32"
	case appdef.DataKind_int64:
		return "int64"
	case appdef.DataKind_float32:
		return "float32"
	case appdef.DataKind_float64:
		return "float64"
	case appdef.DataKind_bytes:
		return "bytes"
	case appdef.DataKind_string:
		return "string"
	default:
		return unknownFieldType
	}
}

// Custom function to capitalize the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
