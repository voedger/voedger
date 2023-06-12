/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package extensionpoints

import "fmt"

func NewRootExtensionPoint() IExtensionPoint {
	return &implIExtensionPoint{}
}

func (ep *implIExtensionPoint) ExtensionPoint(eKey EKey, value ...interface{}) IExtensionPoint {
	if len(value) > 1 {
		panic("value len must be 0 or 1")
	}
	var res *implIExtensionPoint
	intf, ok := ep.Find(eKey)
	if ok {
		res, ok = intf.(*implIExtensionPoint)
		if !ok {
			panic("already have non extension point under key " + fmt.Sprint(eKey))
		}
	} else {
		res = &implIExtensionPoint{key: eKey}
		ep.exts = append(ep.exts, res)
	}
	if len(value) != 0 {
		if res.value != nil {
			panic("value is set already for extension point " + fmt.Sprint(eKey))
		}
		res.value = value[0]
	}
	return res
}

func (ep *implIExtensionPoint) Find(eKey EKey) (value interface{}, ok bool) {
	for _, extIntf := range ep.exts {
		switch ext := extIntf.(type) {
		case NamedExtension:
			if ext.key == eKey {
				return ext.value, true
			}
		case *implIExtensionPoint:
			if ext.key == eKey {
				return IExtensionPoint(ext), true
			}
		default:
			if ext == eKey {
				return ext, true
			}
		}
	}
	return nil, false
}

func (ep *implIExtensionPoint) AddNamed(eKey EKey, value interface{}) {
	if _, ok := ep.Find(eKey); ok {
		panic(fmt.Sprint(eKey) + " already added")
	}
	ep.exts = append(ep.exts, NamedExtension{key: eKey, value: value})
}

func (ep *implIExtensionPoint) Add(value interface{}) {
	if _, ok := ep.Find(value); ok {
		panic(fmt.Sprint(value) + " already added")
	}
	ep.exts = append(ep.exts, value)
}

func (ep *implIExtensionPoint) Iterate(callback func(eKey EKey, value interface{})) {
	for _, v := range ep.exts {
		switch ext := v.(type) {
		case NamedExtension:
			callback(ext.key, ext.value)
		case *implIExtensionPoint:
			callback(ext.key, IExtensionPoint(ext))
		default:
			callback(ext, ext)
		}
	}
}

func (ep *implIExtensionPoint) Value() interface{} {
	return ep.value
}
