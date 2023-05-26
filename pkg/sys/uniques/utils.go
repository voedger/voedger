/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

// NullRecordID means no unique or the record is inactive
// unique could be obtained by Uniques.GetForKeySet()
func GetUniqueID(unique istructs.IUnique, appStructs istructs.IAppStructs, doc istructs.IRowReader, wsid istructs.WSID) (istructs.RecordID, error) {
	uniqueKeyValues, err := getUniqueKeyValues(unique, appStructs.AppDef(), doc)
	if err != nil {
		// notest
		return istructs.NullRecordID, err
	}
	return getUniqueIDByValues(uniqueKeyValues, unique, appStructs, wsid)
}

func getUniqueIDByValues(uniqueKeyValues []byte, unique istructs.IUnique, appStructs istructs.IAppStructs, wsid istructs.WSID) (istructs.RecordID, error) {
	kb := appStructs.ViewRecords().KeyBuilder(QNameViewUniques)
	if err := buildUniqueViewKeyByValues(uniqueKeyValues, kb, unique.QName()); err != nil {
		// notest
		return istructs.NullRecordID, err
	}
	val, err := appStructs.ViewRecords().Get(wsid, kb)
	if err == nil {
		return val.AsRecordID(field_ID), nil
	}
	if err == istructsmem.ErrRecordNotFound {
		return istructs.NullRecordID, nil
	}
	return istructs.NullRecordID, err
}
