/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Implements interfaces:
//
//	— istructs.IRecord
//	— istructs.IORecord
//	— istructs.IEditableRecord
//	— istructs.ICRecord
//	— istructs.IGRecord
//	— istructs.IWDocHeadRecord
//	— istructs.ICUDRow
type recordType struct {
	rowType
	isNew bool
}

// makeRecord makes null (appdef.NullQName) record
func makeRecord(appCfg *AppConfigType) recordType {
	rec := recordType{
		rowType: makeRow(appCfg),
	}
	return rec
}

// newRecord create new null (appdef.NullQName) record
func newRecord(appCfg *AppConfigType) *recordType {
	r := makeRecord(appCfg)
	return &r
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
	return rec
}

// istructs.ICUDRow.ModifiedFields
func (row *rowType) ModifiedFields(cb func(appdef.FieldName, interface{}) bool) {
	if row.isActiveModified {
		if !cb(appdef.SystemField_IsActive, row.isActive) {
			return
		}
	}
	row.dyB.IterateFields(nil, func(name string, value interface{}) bool {
		return cb(name, value)
	})
}
