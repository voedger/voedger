/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package extensionpoints

type IExtensionPoint interface {
	// optional value is never set or set once. Otherwise -> panic
	ExtensionPoint(eKey EKey, value ...interface{}) IExtensionPoint
	AddNamed(eKey EKey, value interface{})
	Add(value interface{})
	Find(eKey EKey) (val interface{}, ok bool)
	Iterate(callback func(eKey EKey, value interface{}))
	Value() interface{}
}

type EPKey string
type EKey interface{}

// val could be map[interface{}]interface{} or IExtensionPoint
type implIExtensionPoint struct {
	key   EKey
	exts  []interface{} // element could be any or NamedExtension or IExtensionPoint
	value interface{}
}

type NamedExtension struct {
	key   EKey
	value interface{}
}
