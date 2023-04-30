/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	"reflect"
	"strings"
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

func resolveFuncInSchema(name string, schema *SchemaAST) (function *FunctionStmt) {
	iterate(schema, func(stmt interface{}) {
		if f, ok := stmt.(*FunctionStmt); ok {
			if f.Name == name {
				function = f
			}
		}
	})
	return
}

func isInternalFunc(name DefQName, schema *SchemaAST) bool {
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

func resolveFunc(fn DefQName, srcPkgSchema *PackageSchemaAST, pkgmap map[string]*PackageSchemaAST, cb func(f *FunctionStmt) error) error {
	var targetPkgSch *PackageSchemaAST

	if isInternalFunc(fn, srcPkgSchema.Ast) {
		targetPkgSch = srcPkgSchema
	} else {
		pkgQN, err := getQualifiedPackageName(fn.Package, srcPkgSchema.Ast)
		if err != nil {
			return err
		}
		targetPkgSch = pkgmap[pkgQN]
		if targetPkgSch == nil {
			return ErrCouldNotImport(pkgQN)
		}

	}

	f := resolveFuncInSchema(fn.Name, targetPkgSch.Ast)
	if f == nil {
		return ErrUndefined(fn.String())
	}
	return cb(f)

}
