/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
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

func iterateStmt[stmtType *TableStmt | *TypeStmt | *ViewStmt](c IStatementCollection, callback func(stmt stmtType)) {
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

func getQualifiedPackageName(pkgName string, schema *SchemaAST) (string, error) {
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if imp.Alias != nil && *imp.Alias == pkgName {
			return imp.Name, nil
		}
	}
	suffix := fmt.Sprintf("/%s", pkgName)
	for i := 0; i < len(schema.Imports); i++ {
		imp := schema.Imports[i]
		if strings.HasSuffix(imp.Name, suffix) {
			return imp.Name, nil
		}
	}
	return "", ErrUndefined(pkgName)
}

func getTargetSchema(n DefQName, c *basicContext) (*PackageSchemaAST, error) {
	var targetPkgSch *PackageSchemaAST

	if isInternalName(n, c.pkg.Ast) {
		return c.pkg, nil
	}

	pkgQN, err := getQualifiedPackageName(n.Package, c.pkg.Ast)
	if err != nil {
		return nil, err
	}
	targetPkgSch = c.pkgmap[pkgQN]
	if targetPkgSch == nil {
		return nil, ErrCouldNotImport(pkgQN)
	}
	return targetPkgSch, nil
}

func resolveTable(fn DefQName, c *basicContext, pos *lexer.Position) (*TableStmt, error) {
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
		return nil, errorAt(err, pos)
	}

	iterate(schema.Ast, func(stmt interface{}) {
		checkStatement(stmt)
	})

	if item == nil {
		return nil, errorAt(ErrUndefined(fn.String()), c.pos)
	}

	return item, nil
}

func resolve[stmtType *TableStmt | *TypeStmt | *FunctionStmt | *CommandStmt | *CommentStmt | *RateStmt | *TagStmt](fn DefQName, c *basicContext, cb func(f stmtType) error) {
	schema, err := getTargetSchema(fn, c)
	if err != nil {
		c.errs = append(c.errs, errorAt(err, c.pos))
		return
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
			c.errs = append(c.errs, errorAt(ErrCouldNotImport(appdef.SysPackage), c.pos))
		} else {
			iter(sysSchema.Ast)
		}
	}
	if item == nil {
		c.errs = append(c.errs, errorAt(ErrUndefined(fn.String()), c.pos))
		return
	}
	err = cb(item)
	if err != nil {
		c.errs = append(c.errs, errorAt(err, c.pos))
		return
	}
}

func maybeSysPkg(pkg string) bool {
	return (pkg == "" || pkg == appdef.SysPackage)
}

func isSysDef(qn DefQName, ident string) bool {
	return maybeSysPkg(qn.Package) && qn.Name == ident
}

func isPredefinedSysTable(table *TableStmt, c *buildContext) bool {
	return c.pkg.QualifiedPackageName == appdef.SysPackage &&
		(table.Name == nameCDOC || table.Name == nameWDOC || table.Name == nameODOC ||
			table.Name == nameCRecord || table.Name == nameWRecord || table.Name == nameORecord)
}

func getTableInheritanceChain(table *TableStmt, ctx *buildContext) (chain []DefQName) {
	chain = make([]DefQName, 0)
	var vf func(t *TableStmt)
	vf = func(t *TableStmt) {
		if t.Inherits != nil {
			inherited := *t.Inherits
			resolve(inherited, &ctx.basicContext, func(t *TableStmt) error {
				chain = append(chain, inherited)
				vf(t)
				return nil
			})
		}
	}
	vf(table)
	return chain
}

func getNestedTableKind(rootTableKind appdef.DefKind) appdef.DefKind {
	switch rootTableKind {
	case appdef.DefKind_CDoc:
		return appdef.DefKind_CRecord
	case appdef.DefKind_ODoc:
		return appdef.DefKind_ORecord
	case appdef.DefKind_WDoc:
		return appdef.DefKind_WRecord
	default:
		panic(fmt.Sprintf("unexpected root table kind %d", rootTableKind))
	}
}

// func genUniqueName(tablename string, bc appdef.IUniquesBuilder) string {
// 	tn := strings.ToUpper(tablename)
// 	for i := 1; i < appdef.MaxDefUniqueCount+1; i++ {
// 		un := fmt.Sprintf("%s_UNIQUE%d", tn, i)
// 		if bc.UniqueByName(un) == nil {
// 			return un
// 		}
// 	}
// 	return ""
// }

func getTableDefKind(table *TableStmt, ctx *buildContext) (kind appdef.DefKind, singletone bool) {
	chain := getTableInheritanceChain(table, ctx)
	for _, t := range chain {
		if isSysDef(t, nameCDOC) || isSysDef(t, nameSingleton) {
			return appdef.DefKind_CDoc, isSysDef(t, nameSingleton)
		} else if isSysDef(t, nameODOC) {
			return appdef.DefKind_ODoc, false
		} else if isSysDef(t, nameWDOC) {
			return appdef.DefKind_WDoc, false
		} else if isSysDef(t, nameCRecord) {
			return appdef.DefKind_CRecord, false
		} else if isSysDef(t, nameORecord) {
			return appdef.DefKind_ORecord, false
		} else if isSysDef(t, nameWRecord) {
			return appdef.DefKind_WRecord, false
		}
	}
	return appdef.DefKind_null, false
}

func getTypeDataKind(t TypeQName) appdef.DataKind {
	if maybeSysPkg(t.Package) {
		if t.Name == sysInt32 || t.Name == sysInt {
			return appdef.DataKind_int32
		}
		if t.Name == sysInt64 {
			return appdef.DataKind_int64
		}
		if t.Name == sysFloat32 || t.Name == sysFloat {
			return appdef.DataKind_float32
		}
		if t.Name == sysFloat64 {
			return appdef.DataKind_float64
		}
		if t.Name == sysQName {
			return appdef.DataKind_QName
		}
		if t.Name == sysBool {
			return appdef.DataKind_bool
		}
		if t.Name == sysString {
			return appdef.DataKind_string
		}
		if t.Name == sysBytes {
			return appdef.DataKind_bytes
		}
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
