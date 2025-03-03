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
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func readRecords(wsid istructs.WSID, qName appdef.QName, expr sqlparser.Expr, appStructs istructs.IAppStructs, f *filter,
	callback istructs.ExecQueryCallback, recordID istructs.RecordID) error {
	rr := make([]istructs.RecordGetBatchItem, 0)

	qNameType := appStructs.AppDef().Type(qName)
	isSingleton := false
	if iSingleton, ok := qNameType.(appdef.ISingleton); ok {
		isSingleton = iSingleton.Singleton()
	}

	whereIDs := []istructs.RecordID{}
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
			id, err := parseUint64(compExpr.Right.(*sqlparser.SQLVal).Val)
			if err != nil {
				return err
			}
			whereIDs = append(whereIDs, istructs.RecordID(id))
		case sqlparser.InStr:
			for _, v := range compExpr.Right.(sqlparser.ValTuple) {
				id, err := parseUint64(v.(*sqlparser.SQLVal).Val)
				if err != nil {
					return err
				}
				whereIDs = append(whereIDs, istructs.RecordID(id))
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
			// notest
			return err
		}
		if singletonRec.QName() == appdef.NullQName {
			// singleton queried and it does not exists yet -> return immediately
			return nil
		}
		rr = append(rr, istructs.RecordGetBatchItem{ID: singletonRec.ID()})
	}

	if recordID > 0 {
		rr = append(rr, istructs.RecordGetBatchItem{ID: recordID})
	}

	for _, whereID := range whereIDs {
		rr = append(rr, istructs.RecordGetBatchItem{ID: whereID})
	}

	err := appStructs.Records().GetBatch(wsid, true, rr)
	if err != nil {
		// notest
		return err
	}

	if !f.acceptAll {
		for field := range f.fields {
			if qNameType.(appdef.IWithFields).Field(field) == nil {
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

		data := coreutils.FieldsToMap(r.Record, appStructs.AppDef(), getFilter(f.filter), coreutils.WithAllFields())
		bb, e := json.Marshal(data)
		if e != nil {
			// notest
			return e
		}

		e = callback(&result{value: string(bb)})
		if e != nil {
			return e
		}
	}

	return nil
}
