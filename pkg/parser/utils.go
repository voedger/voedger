/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package sqlschema

import (
	"reflect"

	"github.com/alecthomas/participle/v2/lexer"
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

func CompareParam(pos *lexer.Position, left, right FunctionParam) bool {
	var lt, rt OptQName
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

func CompareParams(pos *lexer.Position, params []FunctionParam, f *FunctionStmt, errs []error) []error {
	if len(params) != len(f.Params) {
		errs = append(errs, errorAt(ErrFunctionParamsIncorrect, pos))
		return errs
	}
	for i := 0; i < len(params); i++ {
		if !CompareParam(pos, params[i], f.Params[i]) {
			errs = append(errs, errorAt(ErrFunctionParamsIncorrect, pos))
		}
	}
	return errs
}

func iterate(c IStatementCollection, callback func(stmt interface{})) {
	c.Iterate(func(stmt interface{}) {
		callback(stmt)
		if collection, ok := stmt.(IStatementCollection); ok {
			iterate(collection, callback)
		}
	})
}
