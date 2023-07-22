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
	var lt, rt TypeQName
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

func iterateStmt[stmtType *TableStmt | *TypeStmt | *ViewStmt | *CommandStmt | *QueryStmt | *WorkspaceStmt](c IStatementCollection, callback func(stmt stmtType)) {
	c.Iterate(func(stmt interface{}) {
		if s, ok := stmt.(stmtType); ok {
			callback(s)
		}
		if collection, ok := stmt.(IStatementCollection); ok {
			iterateStmt(collection, callback)
		}
	})
}

func isInternalName(name DefQName, schema *SchemaAST) bool {
	pkg := strings.TrimSpace(name.Package)
	return pkg == "" || pkg == schema.Package
}

func getQualifiedPackageName(pkgName string, schema *SchemaAST) string {
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

	if isInternalName(n, c.pkg.Ast) {
		return c.pkg, nil
	}

	if n.Package == appdef.SysPackage {
		sysSchema := c.pkgmap[appdef.SysPackage]
		if sysSchema == nil {
			return nil, ErrCouldNotImport(appdef.SysPackage)
		}
		return sysSchema, nil
	}

	pkgQN := getQualifiedPackageName(n.Package, c.pkg.Ast)
	if pkgQN == "" {
		return nil, ErrUndefined(n.Package)
	}
	targetPkgSch = c.pkgmap[pkgQN]
	if targetPkgSch == nil {
		return nil, ErrCouldNotImport(pkgQN)
	}
	return targetPkgSch, nil
}

func resolveTable(fn DefQName, c *basicContext) (*TableStmt, error) {
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
		return nil, err
	}

	iterate(schema.Ast, func(stmt interface{}) {
		checkStatement(stmt)
	})

	if item == nil {
		return nil, ErrUndefined(fn.String())
	}

	return item, nil
}

// when not found, lookup returns (nil, nil)
func lookup[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *RateStmt | *TagStmt |
	*WorkspaceStmt | *ViewStmt | *StorageStmt](fn DefQName, c *basicContext) (stmtType, error) {
	schema, err := getTargetSchema(fn, c)
	if err != nil {
		return nil, err
	}
	var item stmtType
	iter := func(s *SchemaAST) {
		iterate(s, func(stmt interface{}) {
			if f, ok := stmt.(stmtType); ok {
				named := any(f).(INamedStatement)
				if named.GetName() == fn.Name {
					item = f
				}
			}
		})
	}
	iter(schema.Ast)

	if item == nil && maybeSysPkg(fn.Package) { // Look in sys pkg
		sysSchema := c.pkgmap[appdef.SysPackage]
		if sysSchema == nil {
			return nil, ErrCouldNotImport(appdef.SysPackage)
		}
		iter(sysSchema.Ast)
	}

	return item, nil
}

func resolve[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt |
	*RateStmt | *TagStmt | *WorkspaceStmt | *StorageStmt | *ViewStmt](fn DefQName, c *basicContext, cb func(f stmtType) error) error {
	var err error
	var item stmtType
	item, err = lookup[stmtType](fn, c)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrUndefined(fn.String())
	}
	return cb(item)
}

func maybeSysPkg(pkg string) bool {
	return (pkg == "" || pkg == appdef.SysPackage)
}

func isSysDef(qn DefQName, ident string) bool {
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

func isVoid(pkg string, name string) bool {
	if maybeSysPkg(pkg) {
		return name == sysVoid
	}
	return false
}

func getSysDataKind(name string) appdef.DataKind {
	if name == sysInt32 || name == sysInt {
		return appdef.DataKind_int32
	}
	if name == sysInt64 {
		return appdef.DataKind_int64
	}
	if name == sysFloat32 || name == sysFloat {
		return appdef.DataKind_float32
	}
	if name == sysFloat64 {
		return appdef.DataKind_float64
	}
	if name == sysQName {
		return appdef.DataKind_QName
	}
	if name == sysBool {
		return appdef.DataKind_bool
	}
	if name == sysString {
		return appdef.DataKind_string
	}
	if name == sysBytes {
		return appdef.DataKind_bytes
	}
	if name == sysBlob {
		return appdef.DataKind_RecordID
	}
	if name == sysTimestamp {
		return appdef.DataKind_int64
	}
	if name == sysCurrency {
		return appdef.DataKind_int64
	}
	return appdef.DataKind_null
}

func getTypeDataKind(t TypeQName) appdef.DataKind {
	if maybeSysPkg(t.Package) {
		return getSysDataKind(t.Name)
	}
	return appdef.DataKind_null
}

func getDefDataKind(pkg string, name string) appdef.DataKind {
	if maybeSysPkg(pkg) {
		return getSysDataKind(name)
	}
	return appdef.DataKind_null
}

func viewFieldDataKind(f *ViewField) appdef.DataKind {
	if f.Type.Bool {
		return appdef.DataKind_bool
	}
	if f.Type.Bytes {
		return appdef.DataKind_bytes
	}
	if f.Type.Float32 {
		return appdef.DataKind_float32
	}
	if f.Type.Float64 {
		return appdef.DataKind_float64
	}
	if f.Type.Id {
		return appdef.DataKind_RecordID
	}
	if f.Type.Int32 {
		return appdef.DataKind_int32
	}
	if f.Type.Int64 {
		return appdef.DataKind_int64
	}
	if f.Type.QName {
		return appdef.DataKind_QName
	}
	return appdef.DataKind_string
}

func buildQname(ctx *buildContext, pkg string, name string) appdef.QName {
	if pkg == "" {
		pkg = ctx.pkg.Ast.Package
	}
	return appdef.NewQName(pkg, name)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
