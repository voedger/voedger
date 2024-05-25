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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func readRecords(wsid istructs.WSID, qName appdef.QName, expr sqlparser.Expr, appStructs istructs.IAppStructs, f *filter,
	callback istructs.ExecQueryCallback, recordID istructs.RecordID) error {
	rr := make([]istructs.RecordGetBatchItem, 0)

	qNameType := appStructs.AppDef().Type(qName)
	isSingleton := false
	if iSingleton, ok := qNameType.(appdef.ISingleton); ok {
		isSingleton = iSingleton.Singleton()
	}

	// recordID
	// singleton -> recordID == 0 && expr == nil
	// not a singleton -> recordID > 0 -> expr == nil
	//                   recordID == 0 -> expr could be not nil
	whereIDs := []int64{}
	switch compExpr := expr.(type) {
	case nil:
	default:
		return fmt.Errorf("unsupported expression: %T", compExpr)
	case *sqlparser.ComparisonExpr:
		if compExpr.Left.(*sqlparser.ColName).Name.Lowered() != "id" {
			return fmt.Errorf("unsupported column name: %s", compExpr.Left.(*sqlparser.ColName).Name.String())
		}
		switch compExpr.Operator {
		case sqlparser.EqualStr:
			id, err := parseInt64(compExpr.Right.(*sqlparser.SQLVal).Val)
			if err != nil {
				return err
			}
			whereIDs = append(whereIDs, id)
		case sqlparser.InStr:
			if compExpr.Left.(*sqlparser.ColName).Name.Lowered() != "id" {
				return fmt.Errorf("unsupported column name: %s", compExpr.Left.(*sqlparser.ColName).Name.String())
			}
			for _, v := range compExpr.Right.(sqlparser.ValTuple) {
				id, err := parseInt64(v.(*sqlparser.SQLVal).Val)
				if err != nil {
					return err
				}
				whereIDs = append(whereIDs, id)
			}
		default:
			return fmt.Errorf("unsupported operation: %s", compExpr.Operator)
		}
	}

	whereIDProvided := len(whereIDs) > 0
	switch {
	case isSingleton && (recordID > 0 || whereIDProvided):
		return errors.New("conditions are not allowed to query a singleton")
	case !isSingleton && recordID > 0 && whereIDProvided:
		return errors.New("record ID and 'where id ...' clause can not be used in one query")
	case !isSingleton && recordID == 0 && !whereIDProvided:
		return fmt.Errorf("'%s' is not a singleton. At least one record ID must be provided", qName)
	}

	if isSingleton {
		singletonRec, err := appStructs.Records().GetSingleton(wsid, qName)
		if err != nil {
			return err
		}
		if singletonRec.QName() == appdef.NullQName {
			// singleton queried and it does not exists yet -> return immediately
			return nil
		}
		rr = append(rr, istructs.RecordGetBatchItem{ID: singletonRec.ID()})
	}

	// switch r := expr.(type) {
	// case *sqlparser.ComparisonExpr:
	// 	if r.Left.(*sqlparser.ColName).Name.Lowered() != "id" {
	// 		return fmt.Errorf("unsupported column name: %s", r.Left.(*sqlparser.ColName).Name.String())
	// 	}
	// 	if isSingleton {
	// 		return errors.New("conditions are not allowed for singleton")
	// 	}
	// 	if recordID > 0 {
	// 		return errors.New("both record ID and 'where id = ...' in one query is not allowed")
	// 	}
	// 	switch r.Operator {
	// 	case sqlparser.EqualStr:
	// 		id, err := parseInt64(r.Right.(*sqlparser.SQLVal).Val)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		rr = append(rr, istructs.RecordGetBatchItem{ID: istructs.RecordID(id)})
	// 	case sqlparser.InStr:
	// 		if r.Left.(*sqlparser.ColName).Name.Lowered() != "id" {
	// 			return fmt.Errorf("unsupported column name: %s", r.Left.(*sqlparser.ColName).Name.String())
	// 		}
	// 		for _, v := range r.Right.(sqlparser.ValTuple) {
	// 			id, err := parseInt64(v.(*sqlparser.SQLVal).Val)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			rr = append(rr, istructs.RecordGetBatchItem{ID: istructs.RecordID(id)})
	// 		}
	// 	default:
	// 		return fmt.Errorf("unsupported operation: %s", r.Operator)
	// 	}
	// case nil:
	// 	if isSingleton {
	// 		if recordID > 0 {
	// 			return errors.New("record ID is not allowed for singleton")
	// 		}
	// 		singletonRec, e := appStructs.Records().GetSingleton(wsid, qName)
	// 		if e != nil {
	// 			if errors.Is(e, istructsmem.ErrNameNotFound) {
	// 				return fmt.Errorf("'%s' is not a singleton. Please specify at least one record ID", qName)
	// 			}
	// 			return e
	// 		}
	// 		rr = append(rr, istructs.RecordGetBatchItem{ID: singletonRec.ID()})
	// 	} else if recordID == 0 {
	// 		return errors.New("ID must be proivded to query a non-singleton")
	// 	}
	// default:
	// 	return fmt.Errorf("unsupported expression: %T", r)
	// }

	if recordID > 0 && len(rr) == 0 {
		rr = append(rr, istructs.RecordGetBatchItem{ID: recordID})
	}

	err := appStructs.Records().GetBatch(wsid, true, rr)
	if err != nil {
		return err
	}

	if !f.acceptAll {
		for field := range f.fields {
			if qNameType.(appdef.IFields).Field(field) == nil {
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
