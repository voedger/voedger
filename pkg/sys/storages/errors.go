/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

var (
	ErrNotFoundKey                      = errors.New("not found key")
	ErrNotFound                         = errors.New("not found")
	ErrNotSupported                     = errors.New("not supported")
	errNotImplemented                   = errors.New("not implemented")
	errCurrentValueIsNotAnArray         = errors.New("current value is not an array")
	errFieldByIndexIsNotAnObjectOrArray = errors.New("field by index is not an object or array")
	errMockedKeyBuilderExpected         = errors.New("IStataKeyBuilder must be mockedKeyBuilder")
)

func errInt8FieldUndefined(name string) error {
	return errors.New("undefined int8 field: " + name)
}

func errInt16FieldUndefined(name string) error {
	return errors.New("undefined int16 field: " + name)
}

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

func typeIsNotDefinedInWorkspaceWithDescriptor(typ, ws appdef.QName) error {
	return fmt.Errorf("%s %w %s", typ.String(), ErrQNameIsNotDefinedInWorkspace, ws.String())
}

var ErrQNameIsNotDefinedInWorkspace = errors.New("qname is not defined in workspace")

func errUnexpectedType(actual interface{}) error {
	return fmt.Errorf("unexpected type: %v", actual)
}

func errIndexOutOfBounds(index int) error {
	return fmt.Errorf("index out of bounds: %d", index)
}

var errTest = errors.New("test")
var errEntityRequiredForValueBuilder = errors.New("entity required for ValueBuilder")
var errWorkspaceDescriptorNotFound = errors.New("WorkspaceDescriptor not found in workspace")
var errDescriptorForUndefinedWorkspace = errors.New("workspace descriptor for undefined workspace")
var errCommandNotSpecified = errors.New("command not specified")
var errOwnerRecordNotSpecified = errors.New("owner record not specified")
var errOwnerRecordFieldNotSpecified = errors.New("owner record field not specified")
var errOwnerIDNotSpecified = errors.New("owner ID not specified")
