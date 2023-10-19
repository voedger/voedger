/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Returns name of system data type by data kind.
//
// Returns NullQName if data kind is out of bounds.
func sysDataTypeName(k DataKind) QName {
	if (k > DataKind_null) && (k < DataKind_FakeLast) {
		return NewQName(SysPackage, k.TrimString())
	}
	return NullQName
}

// Creates and returns new system type by data kind.
func newSysData(app *appDef, kind DataKind) *data {
	d := &data{}
	d.typ = makeType(app, sysDataTypeName(kind), TypeKind_Data)
	d.dataKind = kind
	app.appendType(d)
	return d
}

// Makes system data types for all data kinds.
//
// Should be called after appDef is created.
func (app *appDef) makeSysDataTypes() {
	for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
		_ = newSysData(app, k)
	}
}
