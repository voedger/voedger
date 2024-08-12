/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

var ErrNotSupported = errors.New("not supported")
var ErrNotFound = errors.New("not found")
var ErrNotExists = errors.New("not exists")
var ErrExists = errors.New("exists")
var ErrIntentsLimitExceeded = errors.New("intents limit exceeded")
var ErrUnknownStorage = errors.New("unknown storage")
var ErrGetNotSupportedByStorage = errors.New("get not supported by storage")
var ErrReadNotSupportedByStorage = errors.New("read not supported by storage")
var ErrUpdateNotSupportedByStorage = errors.New("update not supported by storage")
var ErrInsertNotSupportedByStorage = errors.New("insert not supported by storage")
var errCommandPrepareArgsNotSupportedByState = fmt.Errorf("%w: CommandPrepareArgs available for commands only", errors.ErrUnsupported)
var errQueryPrepareArgsNotSupportedByState = fmt.Errorf("%w: QueryPrepareArgs available for queries only", errors.ErrUnsupported)
var errQueryCallbackNotSupportedByState = fmt.Errorf("%w: QueryCallback available for queries only", errors.ErrUnsupported)
var errTest = errors.New("test")
var errCurrentValueIsNotAnArray = errors.New("current value is not an array")
var errFieldByIndexIsNotAnObjectOrArray = errors.New("field by index is not an object or array")
var errNotImplemented = errors.New("not implemented")
var errEntityRequiredForValueBuilder = errors.New("entity required for ValueBuilder")
var errWorkspaceDescriptorNotFound = errors.New("WorkspaceDescriptor not found in workspace")
var errDescriptorForUndefinedWorkspace = errors.New("workspace descriptor for undefined workspace")
var errCommandNotSpecified = errors.New("command not specified")
var errBlobIDNotSpecified = errors.New("blob ID not specified")
var ErrQNameIsNotDefinedInWorkspace = errors.New("qname is not defined in workspace")

func errUnexpectedType(actual interface{}) error {
	return fmt.Errorf("unexpected type: %v", actual)
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

func errIndexOutOfBounds(index int) error {
	return fmt.Errorf("index out of bounds: %d", index)
}

func typeIsNotDefinedInWorkspaceWithDescriptor(typ, ws appdef.QName) error {
	return fmt.Errorf("%s %w %s", typ.String(), ErrQNameIsNotDefinedInWorkspace, ws.String())
}
