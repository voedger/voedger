/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"slices"
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
	var errList []error
	dir, err := createOrmDir(params.TargetDir)
	if err != nil {
		return err
	}

	headerContent, err := getHeaderFileContent(params.HeaderFile)
	if err != nil {
		errList = append(errList, err)
	}

	pkgObjs, pkgInfos := gatherPackageObjs(compileRes.AppDef, headerContent)
	for pkgLocalName, objs := range pkgObjs {
		if len(objs) == 0 {
			continue
		}
		// generate package_*.go module
		genPkgErrors := generatePackage(pkgLocalName, pkgInfos, objs, dir)
		errList = append(errList, genPkgErrors...)
	}
	// generate types.go module
	if err := generateTypesModule(dir); err != nil {
		errList = append(errList, err)
	}
	return errors.Join(errList...)
}

func gatherPackageObjs(appDef appdef.IAppDef, headerContent string) (map[string][]appdef.IType, map[string]ormPackageInfo) {
	uniqueObjects := make([]string, 0)
	pkgObjs := make(map[string][]appdef.IType)  // mapping of package local names to list of its objects
	pkgInfos := make(map[string]ormPackageInfo) // mapping of package local names to its info
	pkgLocalNames := appDef.PackageLocalNames()
	for _, pkgLocalName := range pkgLocalNames {
		pkgObjs[pkgLocalName] = make([]appdef.IType, 0)
	}

	collectObjectsFunc := func(iType appdef.IType) {
		if _, ok := iType.(appdef.IWorkspace); ok {
			return
		}
		qName := iType.QName()
		pkgLocalName := qName.Pkg()
		if !slices.Contains(uniqueObjects, qName.String()) {
			pkgObjs[pkgLocalName] = append(pkgObjs[pkgLocalName], iType)
			uniqueObjects = append(uniqueObjects, qName.String())
		}
	}

	appDef.Types(func(iType appdef.IType) {
		pkgLocalName := iType.QName().Pkg()
		// if the object is in the system package, we collect it
		if pkgLocalName == appdef.SysPackage {
			collectObjectsFunc(iType)
			return
		}
		// for other packages we collect objects inside the workspaces
		if workspace, ok := iType.(appdef.IWorkspace); ok {
			workspace.Types(collectObjectsFunc)
		}
	})

	for pkgLocalName := range pkgObjs {
		pkgFullPath := appDef.PackageFullPath(pkgLocalName)
		if pkgFullPath == "" {
			pkgFullPath = appdef.SysPackage
		}
		pkgInfos[pkgLocalName] = ormPackageInfo{
			Name:              pkgLocalName,
			FullPath:          pkgFullPath,
			HeaderFileContent: headerContent,
		}
	}
	return pkgObjs, pkgInfos
}

func generatePackage(pkgLocalName string, pkgInfos map[string]ormPackageInfo, objs []appdef.IType, dir string) []error {
	var errList []error
	pkgData, errs := fillPackageData(pkgLocalName, pkgInfos, objs)
	errList = append(errList, errs...)

	pkgFile, err := fillTemplate(pkgData)
	if err != nil {
		errList = append(errList, err)
	}

	filePath := filepath.Join(dir, fmt.Sprintf("package_%s.go", pkgLocalName))
	if err := saveFile(filePath, pkgFile); err != nil {
		errList = append(errList, err)
	}
	return errList
}

func generateTypesModule(dir string) error {
	sysFilePath := filepath.Join(dir, "types.go")
	return saveFile(sysFilePath, []byte(sysContent))
}

func saveFile(filePath string, content []byte) error {
	return os.WriteFile(filePath, content, defaultPermissions)
}

func fillPackageData(pkgLocalName string, pkgInfos map[string]ormPackageInfo, objs []appdef.IType) (ormPackage, []error) {
	var errList []error
	pkgData := ormPackage{
		ormPackageInfo: pkgInfos[pkgLocalName],
		Imports:        []string{exttinygoImport},
		Items:          make([]interface{}, 0),
	}
	for _, obj := range objs {
		item := getPackageItem(pkgLocalName, pkgInfos, obj)
		if item != nil {
			pkgData.Items = append(pkgData.Items, item)
		}
	}
	return pkgData, errList
}

func newPackageItem(defaultPkgLocalName string, pkgInfos map[string]ormPackageInfo, obj interface{}) ormPackageItem {
	name := getName(obj)
	pkgLocalName := defaultPkgLocalName
	if obj != nil {
		pkgLocalName = obj.(appdef.IType).QName().Pkg()
	}
	pkgInfo := pkgInfos[pkgLocalName]
	return ormPackageItem{
		Package:    pkgInfo,
		TypeQName:  fmt.Sprintf("%s.%s", pkgInfo.FullPath, name),
		Name:       name,
		Type:       getObjType(obj),
		SqlContent: "Here will be SQL content of the table.",
	}
}

func newFieldItem(tableData ormTableItem, field appdef.IField) ormField {
	name := normalizeName(field.Name())
	return ormField{
		Table:         tableData,
		Type:          getFieldType(field),
		Name:          normalizeName(field.Name()),
		GetMethodName: fmt.Sprintf("Get_%s", strings.ToLower(name)),
		SetMethodName: fmt.Sprintf("Set_%s", strings.ToLower(name)),
	}
}

func getPackageItem(pkgLocalName string, pkgInfos map[string]ormPackageInfo, obj appdef.IType) interface{} {
	if obj == nil {
		return nil
	}
	pkgItem := newPackageItem(pkgLocalName, pkgInfos, obj)
	if pkgItem.Type == unknownType {
		return nil
	}
	switch t := obj.(type) {
	case appdef.ICDoc, appdef.IWDoc, appdef.IView, appdef.IODoc, appdef.IObject:
		tableData := ormTableItem{
			ormPackageItem: pkgItem,
			Fields:         make([]ormField, 0),
		}

		iView, isView := t.(appdef.IView)
		if isView {
			for _, key := range iView.Key().Fields() {
				fieldItem := newFieldItem(tableData, key)
				if fieldItem.Type == unknownType {
					continue
				}
				tableData.Keys = append(tableData.Keys, fieldItem)
			}
		}
		// fetching fields
		for _, field := range t.(appdef.IFields).Fields() {
			fieldItem := newFieldItem(tableData, field)
			if fieldItem.Type == unknownType {
				continue
			}

			isKey := false
			for _, key := range tableData.Keys {
				if key.Name == fieldItem.Name {
					isKey = true
					break
				}
			}
			if !isKey {
				tableData.Fields = append(tableData.Fields, fieldItem)
			}
		}
		return tableData
	case appdef.ICommand, appdef.IQuery:
		var resultFields []ormField
		argumentObj := getPackageItem(pkgLocalName, pkgInfos, t.(appdef.IFunction).Param())

		var unloggedArgumentObj interface{}
		if iCommand, ok := t.(appdef.ICommand); ok {
			unloggedArgumentObj = getPackageItem(pkgLocalName, pkgInfos, iCommand.UnloggedParam())
		}
		if resultObj := getPackageItem(pkgLocalName, pkgInfos, t.(appdef.IFunction).Result()); resultObj != nil {
			if tableData, ok := resultObj.(ormTableItem); ok {
				resultFields = tableData.Fields
			}
		}

		commandItem := ormCommand{
			ormPackageItem:         pkgItem,
			ArgumentObject:         argumentObj,
			ResultObjectFields:     resultFields,
			UnloggedArgumentObject: unloggedArgumentObj,
		}
		return commandItem
	default:
		typeKind := t.Kind()
		if typeKind == appdef.TypeKind_Object {
			return getPackageItem(pkgLocalName, pkgInfos, t.(appdef.IObject))
		}
		return pkgItem
	}
}

func fillTemplate(payload interface{}) ([]byte, error) {
	templatesDir, err := fsTemplates.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}
	templates := make([]string, 0)
	for _, file := range templatesDir {
		templates = append(templates, fmt.Sprintf("templates/%s", file.Name()))
	}

	t, err := template.New("package").Funcs(template.FuncMap{
		"capitalize": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}).ParseFiles(templates...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	var filledTemplate bytes.Buffer
	if err := t.ExecuteTemplate(&filledTemplate, "package", payload); err != nil {
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

func normalizeName(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}

func getName(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return normalizeName(obj.(appdef.IType).QName().Entity())
}

func getObjType(obj interface{}) string {
	switch t := obj.(type) {
	case appdef.IODoc:
		return "ODoc"
	case appdef.ICDoc:
		if t.Singleton() {
			return "CSingleton"
		}
		return "CDoc"
	case appdef.IWDoc:
		if t.Singleton() {
			return "WSingleton"
		}
		return "WDoc"
	case appdef.IView:
		return "View"
	case appdef.ICommand:
		return "Command"
	case appdef.IQuery:
		return "Query"
	case appdef.IObject:
		return getTypeKind(t.Kind())
	case appdef.IType:
		return getTypeKind(t.Kind())
	default:
		return unknownType
	}
}

func getTypeKind(typeKind appdef.TypeKind) string {
	switch typeKind {
	case appdef.TypeKind_Object:
		return "Type"
	case appdef.TypeKind_CDoc:
		return "CDoc"
	case appdef.TypeKind_WDoc:
		return "WDoc"
	case appdef.TypeKind_ODoc:
		return "ODoc"
	default:
		return unknownType
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
		return "Bytes"
	case appdef.DataKind_string:
		return "string"
	case appdef.DataKind_RecordID:
		return "Ref"
	default:
		return unknownType
	}
}
