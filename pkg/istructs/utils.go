/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin AppQName
 * @author: Maxim Geraskin qname_
 */

package istructs

import (
	"encoding/json"
	"strconv"

	"github.com/voedger/voedger/pkg/schemas"
)

// *********************************************************************************************************
//
//				AppQName
//

func NewAppQName(owner, name string) AppQName {
	return AppQName{owner: owner, name: name}
}

func (aqn *AppQName) Owner() string { return aqn.owner }
func (aqn *AppQName) Name() string  { return aqn.name }
func (aqn AppQName) String() string { return aqn.owner + AppQNameQualifierChar + aqn.name }

func ParseAppQName(val string) (res AppQName, err error) {
	s1, s2, err := schemas.ParseQualifiedName(val, AppQNameQualifierChar)
	return NewAppQName(s1, s2), err
}

func (aqn *AppQName) MarshalJSON() ([]byte, error) {
	return json.Marshal(aqn.owner + AppQNameQualifierChar + aqn.name)
}

// need to marshal map[AppQName]any
func (aqn AppQName) MarshalText() (text []byte, err error) {
	js, err := json.Marshal(aqn.owner + AppQNameQualifierChar + aqn.name)
	if err != nil {
		// notest
		return nil, err
	}
	res, err := strconv.Unquote(string(js))
	if err != nil {
		// notest
		return nil, err
	}
	return []byte(res), nil
}

func (aqn *AppQName) UnmarshalJSON(text []byte) (err error) {
	*aqn = AppQName{}
	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	aqn.owner, aqn.name, err = schemas.ParseQualifiedName(str, AppQNameQualifierChar)
	return err
}

// need to unmarshal map[AppQName]any
// golang json looks on UnmarshalText presence only on unmarshal map[QName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[AppQName]any
// see https://github.com/golang/go/issues/29732
func (aqn *AppQName) UnmarshalText(text []byte) (err error) {
	// notest
	return nil
}

// *********************************************************************************************************
//
//				WSID
//

func NewWSID(cluster ClusterID, baseWSID WSID) WSID {
	return WSID(cluster)<<WSIDClusterLShift + baseWSID
}

func NewRecordID(baseID RecordID) RecordID {
	return RecordID(ClusterAsRegisterID)*RegisterFactor + baseID
}

// Used to generate IDs for CDoc/CRecord
func NewCDocCRecordID(baseID RecordID) RecordID {
	return RecordID(ClusterAsCRecordRegisterID)*RegisterFactor + baseID
}

func (wsid WSID) ClusterID() ClusterID {
	return ClusterID(wsid >> WSIDClusterLShift)
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

func (*NullRowReader) AsInt32(name string) int32                                         { return 0 }
func (*NullRowReader) AsInt64(name string) int64                                         { return 0 }
func (*NullRowReader) AsFloat32(name string) float32                                     { return 0 }
func (*NullRowReader) AsFloat64(name string) float64                                     { return 0 }
func (*NullRowReader) AsBytes(name string) []byte                                        { return nil }
func (*NullRowReader) AsString(name string) string                                       { return "" }
func (*NullRowReader) AsRecordID(name string) RecordID                                   { return NullRecordID }
func (*NullRowReader) AsQName(name string) schemas.QName                                 { return schemas.NullQName }
func (*NullRowReader) AsBool(name string) bool                                           { return false }
func (*NullRowReader) RecordIDs(includeNulls bool, cb func(name string, value RecordID)) {}

// Implements IObject
type NullObject struct{ NullRowReader }

func NewNullObject() IObject { return &NullObject{} }

func (*NullObject) QName() schemas.QName                            { return schemas.NullQName }
func (*NullObject) Elements(container string, cb func(el IElement)) {}
func (*NullObject) Containers(cb func(container string))            {}
func (no *NullObject) AsRecord() IRecord                            { return no }
func (no *NullObject) FieldNames(cb func(fieldName string))         {}
func (no *NullObject) Container() string                            { return "" }
func (no *NullObject) ID() RecordID                                 { return NullRecordID }
func (no *NullObject) Parent() RecordID                             { return NullRecordID }

// *********************************************************************************************************
//
//	ResourceKindType
//

func (k ResourceKindType) MarshalText() ([]byte, error) {
	var s string
	if k < ResourceKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
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
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

func ValidatorMatchByQName(cudValidator CUDValidator, cudQName schemas.QName) bool {
	if cudValidator.MatchFunc != nil {
		if cudValidator.MatchFunc(cudQName) {
			return true
		}
	}
	for _, qn := range cudValidator.MatchQNames {
		if qn == cudQName {
			return true
		}
	}
	return false
}
