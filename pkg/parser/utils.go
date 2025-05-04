/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	"reflect"
	"regexp"
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

func resolveInCtx[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *ProjectorStmt | *JobStmt |
	*RateStmt | *TagStmt | *WorkspaceStmt | *StorageStmt | *ViewStmt | *LimitStmt | *QueryStmt | *RoleStmt |
	*DeclareStmt | *WsDescriptorStmt](fn DefQName, ictx *iterateCtx, cb func(f stmtType, schema *PackageSchemaAST) error) error {
	var err error
	var item stmtType
	var p *PackageSchemaAST
	item, p, err = lookupInCtx[stmtType](fn, ictx)
	if err != nil {
		return err
	}

	if item == nil {
		var value interface{} = item
		switch value.(type) {
		case *TableStmt, *WsDescriptorStmt:
			return ErrUndefinedTable(fn)
		case *CommandStmt:
			return ErrUndefinedCommand(fn)
		case *QueryStmt:
			return ErrUndefinedQuery(fn)
		case *TagStmt:
			return ErrUndefinedTag(fn)
		case *RoleStmt:
			return ErrUndefinedRole(fn)
		case *TypeStmt:
			return ErrUndefinedType(fn)
		case *WorkspaceStmt:
			return ErrUndefinedWorkspace(fn)
		case *ProjectorStmt:
			return ErrUndefinedProjector(fn)
		case *RateStmt:
			return ErrUndefinedRate(fn)
		case *ViewStmt:
			return ErrUndefinedView(fn)
		default:
			return ErrUndefined(fn.String())
		}
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

func getCurrentWorkspace(ictx *iterateCtx) workspaceAddr {
	for ic := ictx; ic != nil; ic = ic.parent {
		if aws, isWorkspace := ic.collection.(*AlterWorkspaceStmt); isWorkspace {
			return workspaceAddr{aws.alteredWorkspace, aws.alteredWorkspacePkg}
		}
		if ws, isWorkspace := ic.collection.(*WorkspaceStmt); isWorkspace {
			return workspaceAddr{ws, ic.pkg}
		}
	}
	return workspaceAddr{}
}

func lookupInCtx[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *RateStmt | *TagStmt | *ProjectorStmt | *JobStmt |
	*WorkspaceStmt | *ViewStmt | *StorageStmt | *LimitStmt | *QueryStmt | *RoleStmt | *WsDescriptorStmt | *DeclareStmt](fn DefQName, ictx *iterateCtx) (stmtType, *PackageSchemaAST, error) {
	stmtSchema, err := getTargetSchema(fn, ictx)

	var item stmtType
	var value interface{} = item
	lookInOtherPackages := true
	lookInInheritedWorkspaces := true

	switch value.(type) {
	case *TagStmt:
		lookInOtherPackages = false
		lookInInheritedWorkspaces = false
	}

	if stmtSchema != ictx.pkg && !lookInOtherPackages {
		return nil, nil, nil // do not look tags in other packages
	}

	lookingUpInSchema := stmtSchema

	if err != nil {
		return nil, nil, err
	}

	var schema *PackageSchemaAST = nil
	var lookupCallback func(stmt interface{})
	lookupCallback = func(stmt interface{}) {
		if f, ok := stmt.(stmtType); ok && item == nil {
			named := any(f).(INamedStatement)
			if named.GetName() == string(fn.Name) && lookingUpInSchema == stmtSchema {
				item = f
				schema = lookingUpInSchema
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

	ws := getCurrentWorkspace(ictx)
	// First look in the current workspace
	if ws.workspace != nil {
		ws.workspace.Iterate(lookupCallback)
		if item == nil {
			var value interface{} = item
			if _, ok := value.(*WorkspaceStmt); !ok && lookInInheritedWorkspaces { //  when looking for something else than a workspace, look in the inherited workspaces
				var lookInInherted func(iws *WorkspaceStmt) error
				var chain []*WorkspaceStmt
				lookInInherted = func(iws *WorkspaceStmt) error {
					for _, c := range chain {
						if c == iws {
							return nil // avoid circular references. Note this isn't an error because circular references are analyzed elsewhere
						}
					}
					chain = append(chain, iws)
					for _, dq := range iws.Inherits {
						err := resolveInCtx[*WorkspaceStmt](dq, ictx, func(f *WorkspaceStmt, wSchema *PackageSchemaAST) error {
							if !lookInOtherPackages && wSchema != ictx.pkg {
								return nil // do not look tags in other packages
							}
							if err := lookInInherted(f); err != nil {
								return err
							}
							if item != nil {
								return nil
							}
							lookingUpInSchema = wSchema
							f.Iterate(lookupCallback)
							return nil
						})
						if err != nil {
							return err
						}
					}
					return nil
				}

				err := lookInInherted(ws.workspace)
				if err != nil {
					return nil, nil, err
				}

				if item == nil && lookInOtherPackages {
					sysWorkspace, err := lookupInSysPackage(ictx.basicContext, DefQName{Package: appdef.SysPackage, Name: rootWorkspaceName})
					if err != nil {
						return nil, nil, err
					}
					if sysWorkspace != nil {
						lookingUpInSchema = ictx.app.Packages[appdef.SysPackage]
						sysWorkspace.Iterate(lookupCallback)
					}
				}
			}
		}
	}

	// Look in the package
	if item == nil {
		lookingUpInSchema = stmtSchema
		lookingUpInSchema.Ast.Iterate(lookupCallback)
	}

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
	*WorkspaceStmt | *AlterWorkspaceStmt | *ProjectorStmt | *JobStmt | *RateStmt | *GrantStmt |
	*RevokeStmt | *RoleStmt | *TagStmt | *LimitStmt](pkg *PackageSchemaAST, ctx *basicContext, callback func(stmt stmtType, ctx *iterateCtx)) {
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
				wsCtxs:       ictx.wsCtxs,
			}
			iterateContext(iNestedCtx, callback)
		}
	})
}

func isInternalName(pkgName Ident, pkgAst *PackageSchemaAST) bool {
	pkg := strings.TrimSpace(string(pkgName))
	return pkg == "" || pkg == pkgAst.Name
}

func isIdentifier(input string) bool {
	return regexp.MustCompile("^" + identifierRegexp + "$").MatchString(input)
}

func ExtractLocalPackageName(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func GetQualifiedPackageName(pkgName Ident, schema *SchemaAST) string {
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if imp.Alias != nil && *imp.Alias == pkgName {
			return imp.Name
		}
	}
	suffix := fmt.Sprintf("/%s", pkgName)
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if strings.HasSuffix(imp.Name, suffix) || imp.Name == string(pkgName) {
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

	pkgQN := GetQualifiedPackageName(pnkName, c.pkg.Ast)
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

func getNestedTableKind(rootTableKind appdef.TypeKind) (appdef.TypeKind, error) {
	switch rootTableKind {
	case appdef.TypeKind_CDoc, appdef.TypeKind_CRecord:
		return appdef.TypeKind_CRecord, nil
	case appdef.TypeKind_ODoc, appdef.TypeKind_ORecord:
		return appdef.TypeKind_ORecord, nil
	case appdef.TypeKind_WDoc, appdef.TypeKind_WRecord:
		return appdef.TypeKind_WRecord, nil
	default:
		return appdef.TypeKind_null, ErrUnexpectedRootTableKind(int(rootTableKind))
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
	// [~server.vsql.smallints/cmp.Parser~impl]
	if t.Int8 {
		return appdef.DataKind_int8
	}
	if t.Int16 {
		return appdef.DataKind_int16
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

func contains(s []Identifier, e Ident) bool {
	for _, a := range s {
		if a.Value == e {
			return true
		}
	}
	return false
}
