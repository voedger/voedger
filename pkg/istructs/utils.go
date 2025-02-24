/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package istructs

import (
	"encoding/json"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// *********************************************************************************************************
//
//					WSID
//

//	        clusterID shifted here on <<               zeroed on <<
//	63      62 61 60 59 58 57 ........ 47 46 45 44 43 ..................... 1 0
//
// always 0 └─── ClusterID (16 bits) ───┘ └─────── BaseWSID (47 bits) ────────┘
// casting WSID created by NewWSID() to int64 can not cause data loss since the highest bit is always 0
func NewWSID(cluster ClusterID, baseWSID WSID) WSID {
	if baseWSID > MaxBaseWSID {
		panic("baseWSID overflow")
	}
	return WSID(cluster)<<WSIDClusterLShift + baseWSID
}

func NewRecordID(baseID RecordID) RecordID {
	return RecordID(ClusterAsRegisterID)*RegisterFactor + baseID
}

// Used to generate IDs for CDoc/CRecord
func NewCDocCRecordID(baseID RecordID) RecordID {
	return RecordID(ClusterAsCRecordRegisterID)*RegisterFactor + baseID
}

//	63      62 61 60 59 58 57 ......47 ... 15 14 13 12 11 ..................... 1 0
//
// always 0 └─── ClusterID before ───┘     └──── ClusterID is here after >> ──────┘
func (wsid WSID) ClusterID() ClusterID {
	// data loss could happen on uint16(uint64) cast if bit number 63 is !0
	// bit number 63 is 0 always if WSID was created by NewWSID() -> no data loss here on WSID created by NewWSID()
	return ClusterID(wsid >> WSIDClusterLShift) // nolint G115
}

func (wsid WSID) BaseWSID() WSID {
	return wsid - (WSID(wsid.ClusterID()) << WSIDClusterLShift)
}

// RecordID.IsRaw: returns true if ID is temporary
func (id RecordID) IsRaw() bool {
	return (id >= MinRawRecordID) && (id <= MaxRawRecordID)
}

func (id RecordID) BaseRecordID() RecordID {
	return id % RegisterFactor
}

// Implements IRowReader
type NullRowReader struct{}

func (*NullRowReader) AsInt32(name string) int32        { return 0 }
func (*NullRowReader) AsInt64(name string) int64        { return 0 }
func (*NullRowReader) AsFloat32(name string) float32    { return 0 }
func (*NullRowReader) AsFloat64(name string) float64    { return 0 }
func (*NullRowReader) AsBytes(name string) []byte       { return nil }
func (*NullRowReader) AsString(name string) string      { return "" }
func (*NullRowReader) AsRecordID(name string) RecordID  { return NullRecordID }
func (*NullRowReader) AsQName(name string) appdef.QName { return appdef.NullQName }
func (*NullRowReader) AsBool(name string) bool          { return false }
func (*NullRowReader) RecordIDs(bool) func(func(string, RecordID) bool) {
	return func(func(string, RecordID) bool) {}
}

// Implements IObject
type NullObject struct{ NullRowReader }

func NewNullObject() IObject { return &NullObject{} }

func (*NullObject) QName() appdef.QName                              { return appdef.NullQName }
func (*NullObject) Children(...string) func(func(IObject) bool)      { return func(func(IObject) bool) {} }
func (*NullObject) Containers(func(string) bool)                     {}
func (no *NullObject) AsRecord() IRecord                             { return no }
func (no *NullObject) Fields(func(appdef.IField) bool)               {}
func (no *NullObject) Container() string                             { return "" }
func (no *NullObject) ID() RecordID                                  { return NullRecordID }
func (no *NullObject) Parent() RecordID                              { return NullRecordID }
func (no *NullObject) SpecifiedValues(func(appdef.IField, any) bool) {}

// Implements IRowWriter
type NullRowWriter struct{}

func (*NullRowWriter) PutInt32(string, int32)        {}
func (*NullRowWriter) PutInt64(string, int64)        {}
func (*NullRowWriter) PutFloat32(string, float32)    {}
func (*NullRowWriter) PutFloat64(string, float64)    {}
func (*NullRowWriter) PutBytes(string, []byte)       {}
func (*NullRowWriter) PutString(string, string)      {}
func (*NullRowWriter) PutQName(string, appdef.QName) {}
func (*NullRowWriter) PutBool(string, bool)          {}
func (*NullRowWriter) PutRecordID(string, RecordID)  {}
func (*NullRowWriter) PutNumber(string, json.Number) {}
func (*NullRowWriter) PutChars(string, string)       {}
func (*NullRowWriter) PutFromJSON(map[string]any)    {}

// Implements IObjectBuilder
type NullObjectBuilder struct{ NullRowWriter }

func NewNullObjectBuilder() IObjectBuilder { return &NullObjectBuilder{} }

func (*NullObjectBuilder) FillFromJSON(map[string]any)        {}
func (*NullObjectBuilder) ChildBuilder(string) IObjectBuilder { return NewNullObjectBuilder() }
func (*NullObjectBuilder) Build() (IObject, error)            { return NewNullObject(), nil }

// *********************************************************************************************************
//
//	ResourceKindType
//

func (k ResourceKindType) MarshalText() ([]byte, error) {
	var s string
	if k < ResourceKind_FakeLast {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// *********************************************************************************************************
//
//	RateLimitKind
//

func (k RateLimitKind) MarshalText() ([]byte, error) {
	var s string
	if k < RateLimitKind_FakeLast {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

func (um UnixMilli) String() string {
	return time.Unix(0, int64(um)*int64(time.Millisecond)).Format("2006-01-02 15:04:05 MST")
}
