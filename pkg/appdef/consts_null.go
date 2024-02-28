/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Empty name
const NullName = ""

// Null (empty) QName
var (
	QNameForNull = NewQName(NullName, NullName)
	NullQName    = QNameForNull
)

// NullType is used for return then type is not founded
var NullType = new(nullType)

// NullFields is used for return then IFields is not supported
var NullFields = new(nullFields)

// NullAppDef is IAppDef without any user definitions
var NullAppDef = func() IAppDef {
	adb := New()
	app, _ := adb.Build()
	return app
}()
