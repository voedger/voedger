/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservices

import (
	"reflect"
)

// Extract IService fields of a struct which is supposed to be a result of wire to a map[string]IService
func WiredStructPtrToMap(addresOfWiredStruct interface{}) (res map[string]IService) {

	res = make(map[string]IService)
	val := reflect.ValueOf(addresOfWiredStruct).Elem()

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		service, ok := valueField.Interface().(IService)
		if !ok {
			continue
		}
		res[typeField.Name] = service
	}

	return res
}
