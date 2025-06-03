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

	if !f.acceptAll {
		for field := range f.fields {
			if qNameType.(appdef.IWithFields).Field(field) == nil {
				return fmt.Errorf("field '%s' not found in def", field)
			}
		}
	}

	if len(whereIDs) > 1 {
		// where is not allowed with exact recordID
		return readManyRecords(whereIDs, wsid, qNameType, appStructs, f, callback)
	}
	if len(whereIDs) == 1 {
		return readSingleRecord(whereIDs[0], wsid, appStructs, qNameType, f, callback, isSingleton)
	}

	return readSingleRecord(recordID, wsid, appStructs, qNameType, f, callback, isSingleton)

}

func readManyODocs(ids []istructs.RecordID, wsid istructs.WSID, qName appdef.QName, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback) error {
	for _, id := range ids {
		rec, err := appStructs.Events().GetORec(wsid, id, istructs.NullOffset)
		if err != nil {
			return err
		}
		if err := callbackRec(rec, appStructs, qName, f, callback); err != nil {
			return err
		}
	}
	return nil
}

func readManyRecords(ids []istructs.RecordID, wsid istructs.WSID, qNameType appdef.IType, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback) error {
	if qNameType.Kind() == appdef.TypeKind_ODoc || qNameType.Kind() == appdef.TypeKind_ORecord {
		return readManyODocs(ids, wsid, qNameType.QName(), appStructs, f, callback)
	}
	rr := make([]istructs.RecordGetBatchItem, 0)
	for _, whereID := range ids {
		rr = append(rr, istructs.RecordGetBatchItem{ID: whereID})
	}

	err := appStructs.Records().GetBatch(wsid, true, rr)
	if err != nil {
		// notest
		return err
	}

	for _, r := range rr {
		if err := callbackRec(r.Record, appStructs, r.Record.QName(), f, callback); err != nil {
			return err
		}
	}
	return nil
}

func callbackRec(rec istructs.IRecord, appStructs istructs.IAppStructs, qName appdef.QName, f *filter, callback istructs.ExecQueryCallback) error {
	if rec.QName() == appdef.NullQName {
		return fmt.Errorf("record with ID '%d' not found", rec.ID())
	}
	if rec.QName() != qName {
		return fmt.Errorf("record with ID '%d' has mismatching QName '%s'", rec.ID(), rec.QName())
	}

	data := coreutils.FieldsToMap(rec, appStructs.AppDef(), getFilter(f.filter), coreutils.WithAllFields())
	bb, err := json.Marshal(data)
	if err != nil {
		// notest
		return err
	}

	return callback(&result{value: string(bb)})
}

func readSingleRecord(id istructs.RecordID, wsid istructs.WSID, appStructs istructs.IAppStructs,
	qNameType appdef.IType, f *filter, callback istructs.ExecQueryCallback, isSingleton bool) (err error) {
	if isSingleton {
		return readSingleton(wsid, appStructs, qNameType.QName(), f, callback)
	}

	var rec istructs.IRecord
	if qNameType.Kind() == appdef.TypeKind_ODoc || qNameType.Kind() == appdef.TypeKind_ORecord {
		rec, err = appStructs.Events().GetORec(wsid, id, istructs.NullOffset)
	} else {
		rec, err = appStructs.Records().Get(wsid, true, id)
	}
	if err != nil {
		// notest
		return err
	}

	return callbackRec(rec, appStructs, qNameType.QName(), f, callback)
}

func readSingleton(wsid istructs.WSID, appStructs istructs.IAppStructs,
	qName appdef.QName, f *filter, callback istructs.ExecQueryCallback) error {
	singletonRec, err := appStructs.Records().GetSingleton(wsid, qName)
	if err != nil {
		// notest
		return err
	}
	return callbackRec(singletonRec, appStructs, qName, f, callback)
}
