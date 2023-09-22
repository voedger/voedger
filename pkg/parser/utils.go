/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

func extractStatement(s any) interface{} {
	v := reflect.ValueOf(s)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			return field.Interface()
		}
	}
	panic("undefined statement")
}

func CompareParam(left, right FunctionParam) bool {
	var lt, rt DataTypeOrDef
	if left.NamedParam != nil {
		lt = left.NamedParam.Type
	} else {
		lt = *left.UnnamedParamType
	}
	if right.NamedParam != nil {
		rt = right.NamedParam.Type
	} else {
		rt = *right.UnnamedParamType
	}
	return lt == rt
}

func CompareParams(params []FunctionParam, f *FunctionStmt) error {
	if len(params) != len(f.Params) {
		return ErrFunctionParamsIncorrect
	}
	for i := 0; i < len(params); i++ {
		if !CompareParam(params[i], f.Params[i]) {
			return ErrFunctionParamsIncorrect
		}
	}
	return nil
}

func iterate(c IStatementCollection, callback func(stmt interface{})) {
	c.Iterate(func(stmt interface{}) {
		callback(stmt)
		if collection, ok := stmt.(IStatementCollection); ok {
			iterate(collection, callback)
		}
	})
}

func iterateStmt[stmtType *TableStmt | *TypeStmt | *ViewStmt | *CommandStmt | *QueryStmt |
	*WorkspaceStmt | *AlterWorkspaceStmt](c IStatementCollection, callback func(stmt stmtType)) {
	c.Iterate(func(stmt interface{}) {
		if s, ok := stmt.(stmtType); ok {
			callback(s)
		}
		if collection, ok := stmt.(IStatementCollection); ok {
			iterateStmt(collection, callback)
		}
	})
}

func isInternalName(name DefQName, pkgAst *PackageSchemaAST) bool {
	pkg := strings.TrimSpace(string(name.Package))
	return pkg == "" || pkg == string(pkgAst.Name)
}

func getPackageName(pkgQN string) string {
	parts := strings.Split(pkgQN, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func getQualifiedPackageName(pkgName Ident, schema *SchemaAST) string {
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if imp.Alias != nil && *imp.Alias == pkgName {
			return imp.Name
		}
	}
	suffix := fmt.Sprintf("/%s", pkgName)
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if strings.HasSuffix(imp.Name, suffix) {
			return imp.Name
		}
	}
	return ""
}

func getTargetSchema(n DefQName, c *basicContext) (*PackageSchemaAST, error) {
	var targetPkgSch *PackageSchemaAST

	if isInternalName(n, c.pkg) {
		return c.pkg, nil
	}

	if n.Package == appdef.SysPackage {
		sysSchema := c.app.Packages[appdef.SysPackage]
		if sysSchema == nil {
			return nil, ErrCouldNotImport(appdef.SysPackage)
		}
		return sysSchema, nil
	}

	pkgQN := getQualifiedPackageName(n.Package, c.pkg.Ast)
	if pkgQN == "" {
		return nil, ErrUndefined(string(n.Package))
	}
	targetPkgSch = c.app.Packages[pkgQN]
	if targetPkgSch == nil {
		return nil, ErrCouldNotImport(pkgQN)
	}
	return targetPkgSch, nil
}

func resolveTable(fn DefQName, c *basicContext) (*TableStmt, *PackageSchemaAST, error) {
	var item *TableStmt
	var checkStatement func(stmt interface{})
	checkStatement = func(stmt interface{}) {
		if t, ok := stmt.(*TableStmt); ok {
			if t.Name == fn.Name {
				item = t
				return
			}
			for i := range t.Items {
				if t.Items[i].NestedTable != nil {
					checkStatement(&t.Items[i].NestedTable.Table)
				}
			}
		}
	}

	schema, err := getTargetSchema(fn, c)
	if err != nil {
		return nil, nil, err
	}

	iterate(schema.Ast, func(stmt interface{}) {
		checkStatement(stmt)
	})

	if item == nil {
		return nil, nil, ErrUndefined(fn.String())
	}

	return item, schema, nil
}

// when not found, lookup returns (nil, ?, nil)
func lookup[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *RateStmt | *TagStmt |
	*WorkspaceStmt | *ViewStmt | *StorageStmt](fn DefQName, c *basicContext) (stmtType, *PackageSchemaAST, error) {
	schema, err := getTargetSchema(fn, c)
	if err != nil {
		return nil, nil, err
	}
	var item stmtType
	iter := func(s *SchemaAST) {
		iterate(s, func(stmt interface{}) {
			if f, ok := stmt.(stmtType); ok {
				named := any(f).(INamedStatement)
				if named.GetName() == string(fn.Name) {
					item = f
				}
			}
		})
	}
	iter(schema.Ast)

	if item == nil && maybeSysPkg(fn.Package) { // Look in sys pkg
		schema = c.app.Packages[appdef.SysPackage]
		if schema == nil {
			return nil, nil, ErrCouldNotImport(appdef.SysPackage)
		}
		iter(schema.Ast)
	}

	return item, schema, nil
}

func resolve[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt |
	*RateStmt | *TagStmt | *WorkspaceStmt | *StorageStmt | *ViewStmt](fn DefQName, c *basicContext, cb func(f stmtType) error) error {
	var err error
	var item stmtType
	item, _, err = lookup[stmtType](fn, c)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrUndefined(fn.String())
	}
	return cb(item)
}

func resolveEx[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt |
	*RateStmt | *TagStmt | *WorkspaceStmt | *StorageStmt | *ViewStmt](fn DefQName, c *basicContext, cb func(f stmtType, schema *PackageSchemaAST) error) error {
	var err error
	var item stmtType
	var schema *PackageSchemaAST
	item, schema, err = lookup[stmtType](fn, c)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrUndefined(fn.String())
	}
	return cb(item, schema)
}

func maybeSysPkg(pkg Ident) bool {
	return (pkg == "" || pkg == appdef.SysPackage)
}

func isSysDef(qn DefQName, ident Ident) bool {
	return maybeSysPkg(qn.Package) && qn.Name == ident
}

func isPredefinedSysTable(packageName string, table *TableStmt) bool {
	return packageName == appdef.SysPackage &&
		(table.Name == nameCDOC || table.Name == nameWDOC || table.Name == nameODOC ||
			table.Name == nameCRecord || table.Name == nameWRecord || table.Name == nameORecord)
}

func getNestedTableKind(rootTableKind appdef.DefKind) appdef.DefKind {
	switch rootTableKind {
	case appdef.DefKind_CDoc, appdef.DefKind_CRecord:
		return appdef.DefKind_CRecord
	case appdef.DefKind_ODoc, appdef.DefKind_ORecord:
		return appdef.DefKind_ORecord
	case appdef.DefKind_WDoc, appdef.DefKind_WRecord:
		return appdef.DefKind_WRecord
	default:
		panic(fmt.Sprintf("unexpected root table kind %d", rootTableKind))
	}
}

func dataTypeToDataKind(t DataType) appdef.DataKind {
	if t.Blob {
		return appdef.DataKind_RecordID
	}
	if t.Bool {
		return appdef.DataKind_bool
	}
	if t.Bytes != nil {
		return appdef.DataKind_bytes
	}
	if t.Currency {
		return appdef.DataKind_int64
	}
	if t.Float32 {
		return appdef.DataKind_float32
	}
	if t.Float64 {
		return appdef.DataKind_float64
	}
	if t.Int32 {
		return appdef.DataKind_int32
	}
	if t.Int64 {
		return appdef.DataKind_int64
	}
	if t.QName {
		return appdef.DataKind_QName
	}
	if t.Varchar != nil {
		return appdef.DataKind_string
	}
	if t.Timestamp {
		return appdef.DataKind_int64
	}
	return appdef.DataKind_null
}

func buildQname(ctx *buildContext, pkg Ident, name Ident) appdef.QName {
	if pkg == "" {
		pkg = Ident(ctx.pkg.Name)
	}
	return appdef.NewQName(string(pkg), string(name))
}

func contains(s []Ident, e Ident) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
