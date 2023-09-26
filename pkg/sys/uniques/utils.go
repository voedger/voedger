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
// Unique could be obtained by appStructs.AppDef().Type(doc.QName()).Uniques().UniqueField()
func GetUniqueID(appStructs istructs.IAppStructs, doc istructs.IRowReader, wsid istructs.WSID) (recID istructs.RecordID, err error) {
	qName := doc.AsQName(appdef.SystemField_QName)
	if uniques, ok := appStructs.AppDef().Type(qName).(appdef.IUniques); ok {
		if field := uniques.UniqueField(); field != nil {
			uniqueKeyValues, err := getUniqueKeyValues(doc, field)
			if err != nil {
				// notest
				return istructs.NullRecordID, err
			}
			recID, _, err = getUniqueIDByValues(appStructs, wsid, qName, uniqueKeyValues)
		}
	}
	return recID, err
}

func getUniqueIDByValues(appStructs istructs.IAppStructs, wsid istructs.WSID, qName appdef.QName, uniqueKeyValues []byte) (istructs.RecordID, bool, error) {
	kb := appStructs.ViewRecords().KeyBuilder(QNameViewUniques)
	if err := buildUniqueViewKeyByValues(kb, qName, uniqueKeyValues); err != nil {
		// notest
		return istructs.NullRecordID, false, err
	}
	val, err := appStructs.ViewRecords().Get(wsid, kb)
	if err == nil {
		return val.AsRecordID(field_ID), true, nil
	}
	if err == istructsmem.ErrRecordNotFound {
		err = nil
	}
	return istructs.NullRecordID, false, err
}
