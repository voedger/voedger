/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import "errors"

var ErrNotSupported = errors.New("not supported")
var errNotImplemented = errors.New("not implemented")
var errCurrentValueIsNotAnArray = errors.New("current value is not an array")
var errFieldByIndexIsNotAnObjectOrArray = errors.New("field by index is not an object or array")

func errInt32FieldUndefined(name string) error {
	return errors.New("undefined int32 field: " + name)
}

func errInt64FieldUndefined(name string) error {
	return errors.New("undefined int64 field: " + name)
}

func errFloat32FieldUndefined(name string) error {
	return errors.New("undefined float32 field: " + name)
}

func errFloat64FieldUndefined(name string) error {
	return errors.New("undefined float64 field: " + name)
}

func errBoolFieldUndefined(name string) error {
	return errors.New("undefined bool field: " + name)
}

func errStringFieldUndefined(name string) error {
	return errors.New("undefined string field: " + name)
}

func errBytesFieldUndefined(name string) error {
	return errors.New("undefined bytes field: " + name)
}

func errQNameFieldUndefined(name string) error {
	return errors.New("undefined QName field: " + name)
}

func errRecordIDFieldUndefined(name string) error {
	return errors.New("undefined RecordID field: " + name)
}

func errNumberFieldUndefined(name string) error {
	return errors.New("undefined number field: " + name)
}

func errCharsFieldUndefined(name string) error {
	return errors.New("undefined chars field: " + name)
}

func errValueFieldUndefined(name string) error {
	return errors.New("undefined value field: " + name)
}
