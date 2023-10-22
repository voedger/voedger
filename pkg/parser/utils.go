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

func resolveInCtx[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *ProjectorStmt |
	*RateStmt | *TagStmt | *WorkspaceStmt | *StorageStmt | *ViewStmt](fn DefQName, ictx *iterateCtx, cb func(f stmtType, schema *PackageSchemaAST) error) error {
	var err error
	var item stmtType
	var p *PackageSchemaAST
	item, p, err = lookupInCtx[stmtType](fn, ictx)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrUndefined(fn.String())
	}
	return cb(item, p)
}

func lookupInSysPackage[stmtType *WorkspaceStmt](ctx *basicContext, fn DefQName) (stmtType, error) {
	sysSchema := ctx.app.Packages[appdef.SysPackage]
	if sysSchema == nil {
		return nil, ErrCouldNotImport(appdef.SysPackage)
	}
	ictx := &iterateCtx{
		basicContext: ctx,
		collection:   sysSchema.Ast,
		pkg:          sysSchema,
		parent:       nil,
	}
	s, _, e := lookupInCtx[stmtType](fn, ictx)
	return s, e
}

func lookupInCtx[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *RateStmt | *TagStmt | *ProjectorStmt |
	*WorkspaceStmt | *ViewStmt | *StorageStmt](fn DefQName, ictx *iterateCtx) (stmtType, *PackageSchemaAST, error) {
	schema, err := getTargetSchema(fn, ictx)
	if err != nil {
		return nil, nil, err
	}

	var item stmtType
	var lookupCallback func(stmt interface{})
	lookupCallback = func(stmt interface{}) {
		if f, ok := stmt.(stmtType); ok && item == nil {
			named := any(f).(INamedStatement)
			if named.GetName() == string(fn.Name) {
				item = f
			}
		}
		if collection, ok := stmt.(IStatementCollection); ok && item == nil {
			if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace { // do not go into workspaces
				collection.Iterate(lookupCallback)
			}
		}
		if t, ok := stmt.(*TableStmt); ok && item == nil {
			for i := range t.Items {
				if t.Items[i].NestedTable != nil {
					lookupCallback(&t.Items[i].NestedTable.Table)
				}
			}
		}
	}

	if schema == ictx.pkg {

		// Am I in a workspace?
		var ic *iterateCtx = ictx
		var ws *WorkspaceStmt = nil
		for ic != nil && ws == nil {
			if _, isWorkspace := ic.collection.(*WorkspaceStmt); isWorkspace {
				ws = ic.collection.(*WorkspaceStmt)
				break
			}
			ic = ic.parent
		}
		// First look in the current workspace
		if ws != nil {
			ws.Iterate(lookupCallback)
		}

		// Look in the package
		if item == nil {
			schema.Ast.Iterate(lookupCallback)
		}

		// Look in the sys package
		if item == nil && maybeSysPkg(fn.Package) { // Look in sys pkg
			schema = ictx.app.Packages[appdef.SysPackage]
			if schema == nil {
				return nil, nil, ErrCouldNotImport(appdef.SysPackage)
			}
			iterPkg := func(coll IStatementCollection) {
				coll.Iterate(lookupCallback)
			}
			iterPkg(schema.Ast)
		}
		return item, schema, nil
	}

	schema.Ast.Iterate(lookupCallback)
	return item, schema, nil
}

func iteratePackage(pkg *PackageSchemaAST, ctx *basicContext, callback func(stmt interface{}, ctx *iterateCtx)) {
	ictx := &iterateCtx{
		basicContext: ctx,
		collection:   pkg.Ast,
		pkg:          pkg,
		parent:       nil,
	}
	iterateContext(ictx, callback)
}

func iteratePackageStmt[stmtType *TableStmt | *TypeStmt | *ViewStmt | *CommandStmt | *QueryStmt |
	*WorkspaceStmt | *AlterWorkspaceStmt](pkg *PackageSchemaAST, ctx *basicContext, callback func(stmt stmtType, ctx *iterateCtx)) {
	iteratePackage(pkg, ctx, func(stmt interface{}, ctx *iterateCtx) {
		if s, ok := stmt.(stmtType); ok {
			callback(s, ctx)
		}
	})
}

func iterateContext(ictx *iterateCtx, callback func(stmt interface{}, ctx *iterateCtx)) {
	ictx.collection.Iterate(func(stmt interface{}) {
		callback(stmt, ictx)
		if collection, ok := stmt.(IStatementCollection); ok {
			iNestedCtx := &iterateCtx{
				basicContext: ictx.basicContext,
				collection:   collection,
				pkg:          ictx.pkg,
				parent:       ictx,
			}
			iterateContext(iNestedCtx, callback)
		}
	})
}

func isInternalName(pkgName Ident, pkgAst *PackageSchemaAST) bool {
	pkg := strings.TrimSpace(string(pkgName))
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

func findPackage(pnkName Ident, c *iterateCtx) (*PackageSchemaAST, error) {
	var targetPkgSch *PackageSchemaAST
	if isInternalName(pnkName, c.pkg) {
		return c.pkg, nil
	}

	if pnkName == appdef.SysPackage {
		sysSchema := c.app.Packages[appdef.SysPackage]
		if sysSchema == nil {
			return nil, ErrCouldNotImport(appdef.SysPackage)
		}
		return sysSchema, nil
	}

	pkgQN := getQualifiedPackageName(pnkName, c.pkg.Ast)
	if pkgQN == "" {
		return nil, ErrUndefined(string(pnkName))
	}
	targetPkgSch = c.app.Packages[pkgQN]
	if targetPkgSch == nil {
		return nil, ErrCouldNotImport(pkgQN)
	}
	return targetPkgSch, nil

}

func getTargetSchema(n DefQName, c *iterateCtx) (*PackageSchemaAST, error) {
	return findPackage(n.Package, c)
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

func getNestedTableKind(rootTableKind appdef.TypeKind) appdef.TypeKind {
	switch rootTableKind {
	case appdef.TypeKind_CDoc, appdef.TypeKind_CRecord:
		return appdef.TypeKind_CRecord
	case appdef.TypeKind_ODoc, appdef.TypeKind_ORecord:
		return appdef.TypeKind_ORecord
	case appdef.TypeKind_WDoc, appdef.TypeKind_WRecord:
		return appdef.TypeKind_WRecord
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

func buildQname(ctx *iterateCtx, pkg Ident, name Ident) appdef.QName {
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
