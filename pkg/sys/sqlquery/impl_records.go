/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func readRecords(WSID istructs.WSID, qName appdef.QName, expr sqlparser.Expr, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback) error {
	rr := make([]istructs.RecordGetBatchItem, 0)

	findIDs := func(expr sqlparser.Expr) error {
		switch r := expr.(type) {
		case *sqlparser.ComparisonExpr:
			if r.Left.(*sqlparser.ColName).Name.Lowered() != "id" {
				return fmt.Errorf("unsupported column name: %s", r.Left.(*sqlparser.ColName).Name.String())
			}
			switch r.Operator {
			case sqlparser.EqualStr:
				id, err := parseInt64(r.Right.(*sqlparser.SQLVal).Val)
				if err != nil {
					return err
				}
				rr = append(rr, istructs.RecordGetBatchItem{ID: istructs.RecordID(id)})
			case sqlparser.InStr:
				for _, v := range r.Right.(sqlparser.ValTuple) {
					id, err := parseInt64(v.(*sqlparser.SQLVal).Val)
					if err != nil {
						return err
					}
					rr = append(rr, istructs.RecordGetBatchItem{ID: istructs.RecordID(id)})
				}
			default:
				return fmt.Errorf("unsupported operation: %s", r.Operator)
			}
		case nil:
		default:
			return fmt.Errorf("unsupported expression: %T", r)
		}
		return nil
	}
	err := findIDs(expr)
	if err != nil {
		return err
	}

	if expr == nil {
		r, e := appStructs.Records().GetSingleton(WSID, qName)
		if e != nil {
			if errors.Is(e, istructsmem.ErrNameNotFound) {
				return fmt.Errorf("'%s' is not a singleton. Please specify at least one record ID", qName)
			}
			return e
		}
		rr = append(rr, istructs.RecordGetBatchItem{ID: r.ID()})
	}

	if len(rr) == 0 {
		return errors.New("you have to provide at least one record ID")
	}

	err = appStructs.Records().GetBatch(WSID, true, rr)
	if err != nil {
		return err
	}

	def := appStructs.AppDef().Def(qName)
	sf := coreutils.NewFieldsDef(def)

	if !f.acceptAll {
		for field := range f.fields {
			if sf[field] == appdef.DataKind_null {
				return fmt.Errorf("field '%s' not found in def", field)
			}
		}
	}

	for _, r := range rr {
		if r.Record.QName() == appdef.NullQName {
			return fmt.Errorf("record with ID '%d' not found", r.Record.ID())
		}
		if r.Record.QName() != qName {
			return fmt.Errorf("record with ID '%d' has mismatching QName '%s'", r.Record.ID(), r.Record.QName())
		}

		data := coreutils.FieldsToMap(r.Record, appStructs.AppDef(), getFilter(f.filter))
		bb, e := json.Marshal(data)
		if e != nil {
			return e
		}

		e = callback(&result{value: string(bb)})
		if e != nil {
			return e
		}
	}

	return nil
}
