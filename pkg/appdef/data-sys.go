/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Returns name of system data type by data kind.
//
// Returns NullQName if data kind is out of bounds.
func SysDataName(k DataKind) QName {
	if (k > DataKind_null) && (k < DataKind_FakeLast) {
		return NewQName(SysPackage, k.TrimString())
	}
	return NullQName
}

var (
	// System data type names
	SysData_int32    QName = SysDataName(DataKind_int32)
	SysData_int64    QName = SysDataName(DataKind_int64)
	SysData_float32  QName = SysDataName(DataKind_float32)
	SysData_float64  QName = SysDataName(DataKind_float64)
	SysData_bytes    QName = SysDataName(DataKind_bytes)
	SysData_String   QName = SysDataName(DataKind_string)
	SysData_raw      QName = SysDataName(DataKind_raw)
	SysData_QName    QName = SysDataName(DataKind_QName)
	SysData_bool     QName = SysDataName(DataKind_bool)
	SysData_RecordID QName = SysDataName(DataKind_RecordID)
)

// Creates and returns new system type by data kind.
func newSysData(app *appDef, kind DataKind) *data {
	d := &data{
		typ:      makeType(app, SysDataName(kind), TypeKind_Data),
		dataKind: kind,
	}
	app.appendType(d)
	return d
}
