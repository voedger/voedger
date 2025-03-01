/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// # Supports:
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

func NewNullRecord(id istructs.RecordID) istructs.IRecord {
	rec := newRecord(NullAppConfig)
	rec.setID(id)
	return rec
}

// copyFrom assigns record from specified source record
func (rec *recordType) copyFrom(src *recordType) {
	rec.rowType.copyFrom(&src.rowType)
	rec.isNew = src.isNew
}

// return field value by field definition.
//
// # Panics
//   - if unsupported field type
func (row *rowType) fieldValue(f appdef.IField) interface{} {
	n := f.Name()
	switch f.DataKind() {
	case appdef.DataKind_int32:
		return row.AsInt32(n)
	case appdef.DataKind_int64:
		return row.AsInt64(n)
	case appdef.DataKind_float32:
		return row.AsFloat32(n)
	case appdef.DataKind_float64:
		return row.AsFloat64(n)
	case appdef.DataKind_bytes:
		v := row.AsBytes(n)
		if v == nil {
			// #2785
			if _, ok := row.nils[n]; ok {
				v = []byte{}
			}
		}
		return v
	case appdef.DataKind_string:
		return row.AsString(n)
	case appdef.DataKind_QName:
		return row.AsQName(n)
	case appdef.DataKind_bool:
		return row.AsBool(n)
	case appdef.DataKind_RecordID:
		return row.AsRecordID(n)
	case appdef.DataKind_Record:
		return row.AsRecord(n)
	case appdef.DataKind_Event:
		return row.AsEvent(n)
	}
	// notest: fullcase switch
	panic(ErrWrongFieldType("%v", f))
}

// istructs.ICUDRow.IsActivated
func (rec *recordType) IsActivated() bool {
	return !rec.isNew && rec.isActiveModified && rec.isActive
}

// istructs.ICUDRow.IsDeactivated
func (rec *recordType) IsDeactivated() bool {
	return !rec.isNew && rec.isActiveModified && !rec.isActive
}

// istructs.ICUDRow.IsNew
func (rec *recordType) IsNew() bool {
	return rec.isNew
}

// нельзя использовать, если мы хотим прочесть все. т.к. тут зависит от isNew
// нельзя использовать для уникальностей, т.к. там надо именно все поля, которые были реально присланы в запросе

// func (rec *recordType) SpecifiedValues(cb func(appdef.IField, any) bool) {
// 	if rec.isNew {
// 		// если так, то GetCDoc не возвращает id, qname и т.д.
// 		if rec.id != istructs.NullRecordID {
// 			if !cb(rec.fieldDef(appdef.SystemField_ID), rec.id) {
// 				return
// 			}
// 		}
// 		if !cb(rec.fieldDef(appdef.SystemField_IsActive), rec.isActive) {
// 			return
// 		}
// 		if !cb(rec.fieldDef(appdef.SystemField_QName), rec.QName()) {
// 			return
// 		}
// 	}

// 	// если читаем cudRow, у которого isNew, то рендерим все, мы там все задавали, кроме IsActive
// 	// если читаем cudRow,

// 	// а если так, то я бегу по cudRow, где я указал только name =
// 	// if exists, _ := rec.typ.Kind().HasSystemField(appdef.SystemField_ID); exists && rec.id != istructs.NullRecordID {
// 	// 	if !cb(rec.fieldDef(appdef.SystemField_ID), rec.id) {
// 	// 		return
// 	// 	}
// 	// }

// 	// if exists, _ := rec.typ.Kind().HasSystemField(appdef.SystemField_IsActive); exists {
// 	// 	if !rec.isNew {
// 	// 		// а если так, то я получу IsActive даже если я не указал IsActive, но обновляю запись
// 	// 		if !cb(rec.fieldDef(appdef.SystemField_IsActive), rec.isActive) {
// 	// 			return
// 	// 		}
// 	// 	}
// 	// }

// 	if !cb(rec.fieldDef(appdef.SystemField_QName), rec.QName()) {
// 		return
// 	}

// 	rec.dyB.IterateFields(nil, func(name string, value interface{}) bool {
// 		return cb(rec.fieldDef(name), value)
// 	})

// }

// istructs.ICUDRow.SpecifiedValues
// зачем тут думать что я задал или не задал?
// пусть объект сам выдает что у нег выставлено, а что нет


// кароч, так. Тут только те поля, которые реально хранятся. Если сделали Put*, то тут поля не будет
// иными словами, под капотом rowType, который и writer и reader, но SpecifiedFields - это только про хранимые значения,
// а не те,которые Put, но не saved
func (row *rowType) SpecifiedValues(cb func(appdef.IField, any) bool) {
	if row.QName() != appdef.NullQName {
		if !cb(row.fieldDef(appdef.SystemField_QName), row.QName()) {
			return
		}
	}

	if row.id != istructs.NullRecordID {
		if !cb(row.fieldDef(appdef.SystemField_ID), row.id) {
			return
		}
	}

	// if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_ParentID); exists {
	// 	if !cb(row.fieldDef(appdef.SystemField_ParentID), row.parentID) {
	// 		return
	// 	}
	// }
	// if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_Container); exists {
	// 	if !cb(row.fieldDef(appdef.SystemField_Container), row.container) {
	// 		return
	// 	}
	// }

	// if row.isActiveModified {

	// if row.modifiedFields[appdef.SystemField_IsActive] {
	// 	if !cb(row.fieldDef(appdef.SystemField_IsActive), row.isActive) {
	// 		return
	// 	}
	// }

	if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_IsActive); exists {
		if !cb(row.fieldDef(appdef.SystemField_IsActive), row.isActive) {
			return
		}
	}

	// user fields
	row.dyB.IterateFields(nil, func(name string, value interface{}) bool {
		return cb(row.fieldDef(name), value)
	})

}
