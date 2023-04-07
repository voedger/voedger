/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

// recordType implements record that stored in database
//   - interfaces:
//     — istructs.IRecord
//     — istructs.IORecord
//     — istructs.IEditableRecord
//     — istructs.ICRecord
//     — istructs.IGRecord
//     — istructs.IWDocHeadRecord
//     — istructs.ICUDRow
type recordType struct {
	rowType
	isNew bool
}

// newRecord create new null (istructs.NullQName) record
func newRecord(appCfg *AppConfigType) recordType {
	rec := recordType{
		rowType: newRow(appCfg),
	}
	return rec
}

// copyFrom assigns record from specified source record
func (rec *recordType) copyFrom(src *recordType) {
	rec.rowType.copyFrom(&src.rowType)
	rec.isNew = src.isNew
}

// istructs.ICUDRow.IsNew
func (rec *recordType) IsNew() bool {
	return rec.isNew
}

func NewNullRecord(id istructs.RecordID) istructs.IRecord {
	rec := newRecord(NullAppConfig)
	rec.setID(id)
	return &rec
}

// istructs.ICUDRow.ModifiedFields
func (row *rowType) ModifiedFields(cb func(fieldName string, newValue interface{})) {
	row.dyB.IterateFields(nil, func(name string, value interface{}) bool {
		cb(name, value)
		return true
	})
}
