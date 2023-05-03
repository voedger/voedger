/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func readViewRecords(ctx context.Context, WSID istructs.WSID, viewRecordQName appdef.QName, expr sqlparser.Expr, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback) error {
	keyFieldsDef := coreutils.FieldsDef{}
	valueFieldsDef := coreutils.FieldsDef{}

	viewDef := appStructs.AppDef().Def(viewRecordQName)
	viewDef.Containers(func(cont appdef.IContainer) {
		switch cont.Name() {
		case appdef.SystemContainer_ViewPartitionKey, appdef.SystemContainer_ViewClusteringCols:
			appStructs.AppDef().Def(cont.Def()).Fields(func(field appdef.IField) {
				keyFieldsDef[field.Name()] = field.DataKind()
			})
		case appdef.SystemContainer_ViewValue:
			appStructs.AppDef().Def(cont.Def()).Fields(func(field appdef.IField) {
				valueFieldsDef[field.Name()] = field.DataKind()
			})
		}
	})

	if !f.acceptAll {
		allowedFields := make(map[string]bool, len(keyFieldsDef)+len(valueFieldsDef))
		for field := range keyFieldsDef {
			allowedFields[field] = true
		}
		for field := range valueFieldsDef {
			allowedFields[field] = true
		}
		for field := range f.fields {
			if !allowedFields[field] {
				return fmt.Errorf("field '%s' does not exist in '%s' value def", field, viewRecordQName)
			}
		}
	}

	kk := make([]keyPart, 0)
	var keyParts func(expr sqlparser.Expr) error
	keyParts = func(expr sqlparser.Expr) error {
		switch r := expr.(type) {
		case *sqlparser.ComparisonExpr:
			if r.Operator != sqlparser.EqualStr {
				return fmt.Errorf("unsupported operator: %s", r.Operator)
			}

			cn := r.Left.(*sqlparser.ColName)

			var name string
			if !cn.Qualifier.IsEmpty() {
				name = fmt.Sprintf("%s.%s", cn.Qualifier.Name, cn.Name)
			} else {
				name = cn.Name.String()
			}

			kk = append(kk, keyPart{
				name:  name,
				value: r.Right.(*sqlparser.SQLVal).Val,
			})
		case *sqlparser.AndExpr:
			e := keyParts(r.Left)
			if e != nil {
				return e
			}
			e = keyParts(r.Right)
			if e != nil {
				return e
			}
		case nil:
		default:
			return fmt.Errorf("unsupported expression: %T", r)
		}
		return nil
	}
	err := keyParts(expr)
	if err != nil {
		return err
	}

	kb := appStructs.ViewRecords().KeyBuilder(viewRecordQName)

	for _, k := range kk {
		switch keyFieldsDef[k.name] {
		case appdef.DataKind_int32:
			fallthrough
		case appdef.DataKind_int64:
			fallthrough
		case appdef.DataKind_float32:
			fallthrough
		case appdef.DataKind_float64:
			fallthrough
		case appdef.DataKind_RecordID:
			v, e := strconv.ParseFloat(string(k.value), bitSize64)
			if e != nil {
				return e
			}
			kb.PutNumber(k.name, v)
		case appdef.DataKind_bytes:
			fallthrough
		case appdef.DataKind_string:
			fallthrough
		case appdef.DataKind_QName:
			kb.PutChars(k.name, string(k.value))
		case appdef.DataKind_null:
			return fmt.Errorf("field '%s' does not exist in '%s' key def", k.name, viewRecordQName)
		default:
			return errUnsupportedDataKind
		}
	}

	return appStructs.ViewRecords().Read(ctx, WSID, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		data := coreutils.FieldsToMap(key, appStructs.AppDef(), getFilter(f.filter), corecoreutils.WithNonNilsOnly())
		for k, v := range coreutils.FieldsToMap(value, appStructs.AppDef(), getFilter(f.filter), corecoreutils.WithNonNilsOnly()) {
			data[k] = v
		}
		bb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		return callback(&result{value: string(bb)})
	})
}
