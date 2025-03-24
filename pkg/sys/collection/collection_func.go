/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"context"
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func collectionResultQName(args istructs.PrepareArgs) appdef.QName {
	if args.ArgumentObject == nil {
		return appdef.NullQName
	}
	qnameStr := args.ArgumentObject.AsString(Field_Schema)
	qname, err := appdef.ParseQName(qnameStr)
	if err != nil {
		return appdef.NullQName // not provided or incorrect
	}
	return qname
}

func collectionFuncExec(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	if args.ArgumentObject == nil {
		return errors.New("ArgumentObject is not defined in PrepareArgs")
	}
	qnameStr := args.ArgumentObject.AsString(Field_Schema)
	resultsQName, err := appdef.ParseQName(qnameStr)
	if err != nil {
		return err
	}

	kb, err := args.State.KeyBuilder(sys.Storage_View, QNameCollectionView)
	if err != nil {
		return err
	}
	kb.PutInt32(Field_PartKey, PartitionKeyCollection)
	kb.PutQName(Field_DocQName, resultsQName)
	id := args.ArgumentObject.AsRecordID(field_ID)
	if id != istructs.NullRecordID {
		kb.PutRecordID(Field_DocID, id)
	}

	var lastDoc *collectionObject

	err = args.State.Read(kb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
		rec := value.(istructs.IStateViewValue).AsRecord(Field_Record)
		docId := key.AsRecordID(Field_DocID)

		if lastDoc != nil && lastDoc.ID() == docId {
			lastDoc.addRawRecord(rec)
		} else {
			if lastDoc != nil {
				lastDoc.handleRawRecords()
				err = callback(lastDoc)
			}
			obj := newCollectionObject(rec)
			lastDoc = obj
		}
		return
	})
	if lastDoc != nil && err == nil {
		lastDoc.handleRawRecords()
		err = callback(lastDoc)
	}
	return err
}
