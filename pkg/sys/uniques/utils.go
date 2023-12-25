/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// returns ID of the record by the unique combination defined by the doc
// NullRecordID means no records for such unique combination or the record is inactive
// type by doc.QName can not have uniques (e.g. not a table) -> error
func GetUniqueRecordID(appStructs istructs.IAppStructs, doc istructs.IRowReader, wsid istructs.WSID) (recID istructs.RecordID, err error) {
	docQName := doc.AsQName(appdef.SystemField_QName)
	uniques, ok := appStructs.AppDef().Type(docQName).(appdef.IUniques)
	if !ok {
		return istructs.NullRecordID, ErrProvidedDocCanNotHaveUniques
	}
	for _, unique := range uniques.Uniques() {
		schemaOrderedUniqueFields := unique.FieldsSchemaOrdered()
		uniqueKeyValues, err := getUniqueKeyValues(doc, schemaOrderedUniqueFields, unique.Name(), unique.ID())
		if err != nil {
			return istructs.NullRecordID, err
		}
		recID, exists, err := getUniqueIDByValues(appStructs, wsid, docQName, uniqueKeyValues, unique.ID())
		if err != nil {
			return istructs.NullRecordID, err
		}
		if exists {
			return recID, nil
		}
	}
	return istructs.NullRecordID, err
}
