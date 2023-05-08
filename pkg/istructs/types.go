/*
* Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructs

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// AppQName is unique in cluster federation
// <owner>/<name>
// sys/registry
// unTill/airs-bp
// test1/app1
// test1/app2
// test2/app1
// test2/app2
// Ref. utils.go for methods
type AppQName struct {
	owner, name string
}

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

type SubjectKindType uint8

const (
	SubjectKind_null SubjectKindType = iota
	SubjectKind_User
	SubjectKind_Device
	SubjectKind_FakeLast
)

// panics if name does not exist in definition
// If field is nil zero value is returned
type IRowReader interface {
	AsInt32(name string) int32
	AsInt64(name string) int64
	AsFloat32(name string) float32
	AsFloat64(name string) float64
	AsBytes(name string) []byte
	AsString(name string) string
	AsQName(name string) appdef.QName
	AsBool(name string) bool
	AsRecordID(name string) RecordID

	// consts.NullRecord will be returned as null-values
	RecordIDs(includeNulls bool, cb func(name string, value RecordID))
	FieldNames(cb func(fieldName string))
}

type IRowWriter interface {

	// The following functions panics if name has different type then value

	PutInt32(name string, value int32)
	PutInt64(name string, value int64)
	PutFloat32(name string, value float32)
	PutFloat64(name string, value float64)
	PutBytes(name string, value []byte)
	PutString(name, value string)
	PutQName(name string, value appdef.QName)
	PutBool(name string, value bool)
	PutRecordID(name string, value RecordID)

	// Tries to make conversion from value to a name type
	PutNumber(name string, value float64)
	// Tries to make conversion from value to a name type
	PutChars(name string, value string)
}

// App Workspace amount type. Need to wire
type AppWSAmount int
