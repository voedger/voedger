/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"fmt"

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

	if row.parentID != istructs.NullRecordID {
		if !cb(row.fieldDef(appdef.SystemField_ParentID), row.parentID) {
			return
		}
	}

	if len(row.container) > 0 {
		if !cb(row.fieldDef(appdef.SystemField_Container), row.container) {
			return
		}
	}

	if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_IsActive); exists {
		if !cb(row.fieldDef(appdef.SystemField_IsActive), row.isActive) {
			return
		}
	}

	// user fields
	goOn := true
	row.dyB.IterateFields(nil, func(name string, value interface{}) bool {
		field := row.fieldDef(name)
		switch field.DataKind() {
		case appdef.DataKind_RecordID:
			value = istructs.RecordID(value.(int64)) //nolint:gosec
		case appdef.DataKind_QName:
			qNameID := binary.BigEndian.Uint16(value.([]byte))
			qName, err := row.appCfg.qNames.QName(qNameID)
			if err != nil {
				// notest
				panic(err)
			}
			value = qName
		}
		goOn = cb(row.fieldDef(name), value)
		return goOn
	})

	if !goOn {
		return
	}

	for _, nilledIField := range row.nils {
		if !cb(nilledIField, nilToZeroValue(nilledIField.DataKind())) {
			return
		}
	}
}

func nilToZeroValue(kind appdef.DataKind) any {
	switch kind {
	case appdef.DataKind_int32:
		return int32(0)
	case appdef.DataKind_int64:
		return int64(0)
	case appdef.DataKind_float32:
		return float32(0)
	case appdef.DataKind_float64:
		return float64(0)
	case appdef.DataKind_string:
		return ""
	case appdef.DataKind_RecordID:
		return istructs.RecordID(0)
	case appdef.DataKind_QName:
		return appdef.NullQName
	case appdef.DataKind_bool:
		return false
	case appdef.DataKind_bytes:
		return []byte{}
	default:
		panic(fmt.Sprintf("unsupported nilled field kind: %s", kind))
	}
}
