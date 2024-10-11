/*
* Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

type SubjectLogin string

// time.Now().UnixMilli()
type UnixMilli int64

type IDType uint64

// Should be named as ConnectedRegisterID
type ConnectedDeviceID uint16
type PartitionID uint16

type RecordID IDType
type Offset IDType
type WSID IDType

type ClusterID = uint16

// Unique per cluster (Different clusters might have different ID for the same App)
// 2^32 apps per clusters
type ClusterAppID = uint32

type SubjectKindType int32 // not uint8 because it is written to int32 fields, so int32 is better to avoid data loss

const (
	SubjectKind_null SubjectKindType = iota
	SubjectKind_User
	SubjectKind_Device
	SubjectKind_FakeLast
)

// panics if name does not exist in type
// If field is nil zero value is returned
type IRowReader interface {
	AsInt32(appdef.FieldName) int32
	AsInt64(appdef.FieldName) int64
	AsFloat32(appdef.FieldName) float32
	AsFloat64(appdef.FieldName) float64

	// Returns bytes or raw field value
	AsBytes(appdef.FieldName) []byte

	// Returns string or raw field value
	AsString(appdef.FieldName) string

	AsQName(appdef.FieldName) appdef.QName
	AsBool(appdef.FieldName) bool
	AsRecordID(appdef.FieldName) RecordID

	// consts.NullRecord will be returned as null-values
	RecordIDs(includeNulls bool) func(func(appdef.FieldName, RecordID) bool)
	FieldNames(func(appdef.FieldName) bool)
}

type IRowWriter interface {

	// The following functions panics if name has different type then value

	PutInt32(appdef.FieldName, int32)
	PutInt64(appdef.FieldName, int64)
	PutFloat32(appdef.FieldName, float32)
	PutFloat64(appdef.FieldName, float64)

	// Puts value into bytes or raw data field.
	PutBytes(appdef.FieldName, []byte)

	// Puts value into string or raw data field.
	PutString(appdef.FieldName, string)

	PutQName(appdef.FieldName, appdef.QName)
	PutBool(appdef.FieldName, bool)
	PutRecordID(appdef.FieldName, RecordID)

	// Puts underlying json.Number value into field of int32, int64, float32 or float64
	//
	// Tries to make conversion from value to a name type
	PutNumber(appdef.FieldName, json.Number)

	// Puts value into string, bytes or QName data type field.
	//
	// Tries to make conversion from value to a name type
	PutChars(appdef.FieldName, string)

	// Puts value into fields. Field names are taken from map keys, values are taken from map values.
	// types of values and types of the target fields in the schema must be the same
	// joson.Number value type is allowed for number fields
	//
	// Calls PutNumber for numbers and RecordIDs, PutChars for strings, bytes and QNames.
	PutFromJSON(map[appdef.FieldName]any)
}

type NumAppWorkspaces uint16 // since [MaxNumAppWorkspaces] = 32768
type NumAppPartitions uint16 // since [PartitionID] is uint16
type NumCommandProcessors uint
type NumQueryProcessors uint
type AppWorkspaceNumber uint

// RowBuilder is a type for function that creates a row reader from a row writer.
//
// Should return errors.ErrUnsupported if the writer is not supported,
// overwise should build the reader and return it or error.
type RowBuilder func(IRowWriter) (IRowReader, error)

// BuildRow builds a row reader from row writer.
//
// After calling the BuildRow, the writer should not be used,
// as it can lead to changes in the returned reader.
//
// Returns errors.ErrUnsupported if the writer is not supported by known builders.
func BuildRow(w IRowWriter) (IRowReader, error) {
	for _, b := range builders {
		r, err := b(w)
		if err == nil {
			return r, nil
		}
		if !errors.Is(err, errors.ErrUnsupported) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: unknown implementation %#v", errors.ErrUnsupported, w)
}

// CollectRowBuilder collects all row builders.
func CollectRowBuilder(b RowBuilder) bool {
	builders = append(builders, b)
	return true
}

var builders []RowBuilder
