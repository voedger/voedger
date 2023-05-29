/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

// NullRecordID means no unique or the record is inactive
// Unique could be obtained by appStructs.AppDef().Def(doc.QName()).Uniques().UniqueField()
func GetUniqueID(appStructs istructs.IAppStructs, doc istructs.IRowReader, wsid istructs.WSID) (istructs.RecordID, error) {
	qName := doc.AsQName(appdef.SystemField_QName)
	if uniques, ok := appStructs.AppDef().Def(qName).(appdef.IUniques); ok {
		if field := uniques.UniqueField(); field != nil {
			uniqueKeyValues, err := getUniqueKeyValues(doc, field)
			if err != nil {
				// notest
				return istructs.NullRecordID, err
			}
			return getUniqueIDByValues(appStructs, wsid, qName, uniqueKeyValues)
		}
	}
	return istructs.NullRecordID, nil
}

func getUniqueIDByValues(appStructs istructs.IAppStructs, wsid istructs.WSID, qName appdef.QName, uniqueKeyValues []byte) (istructs.RecordID, error) {
	kb := appStructs.ViewRecords().KeyBuilder(QNameViewUniques)
	if err := buildUniqueViewKeyByValues(kb, qName, uniqueKeyValues); err != nil {
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
