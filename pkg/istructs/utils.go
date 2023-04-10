/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin AppQName
 * @author: Maxim Geraskin qname_
 */

package istructs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// *********************************************************************************************************
//
//				qname helpers
//

func qname_parse(val, delimiter string) (part1, part2 string, err error) {
	s := strings.Split(val, delimiter)
	if len(s) != 2 {
		return NullName, NullName, fmt.Errorf("%w: %v", ErrInvalidQNameStringRepresentation, val)
	}
	return s[0], s[1], nil
}

// *********************************************************************************************************
//
//				QName
//

// NewQName: Builds a qualfied name from two parts (from pakage name and from entity name)
func NewQName(pkgName, entityName string) QName {
	return QName{pkg: pkgName, entity: entityName}
}

func ParseQName(val string) (res QName, err error) {
	s1, s2, err := qname_parse(val, QualifierChar)
	return NewQName(s1, s2), err
}

func (qn *QName) Pkg() string    { return qn.pkg }
func (qn *QName) Entity() string { return qn.entity }
func (qn QName) String() string  { return qn.pkg + QualifierChar + qn.entity }

func (qn *QName) MarshalJSON() ([]byte, error) {
	return json.Marshal(qn.pkg + QualifierChar + qn.entity)
}

// need to marshal map[QName]any
func (qn QName) MarshalText() (text []byte, err error) {
	js, err := json.Marshal(qn.pkg + QualifierChar + qn.entity)
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

func (qn *QName) UnmarshalJSON(text []byte) (err error) {
	*qn = QName{}

	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	qn.pkg, qn.entity, err = qname_parse(string(str), QualifierChar)
	return err
}

// need unmarshal map[QName]any
// golang json looks on UnmarshalText presence only on unmarshal map[QName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[QName]any
// see https://github.com/golang/go/issues/29732
func (qn *QName) UnmarshalText(text []byte) (err error) {
	// notest
	return nil
}

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
	s1, s2, err := qname_parse(val, AppQNameQualifierChar)
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
	aqn.owner, aqn.name, err = qname_parse(str, AppQNameQualifierChar)
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
func (*NullRowReader) AsQName(name string) QName                                         { return NullQName }
func (*NullRowReader) AsBool(name string) bool                                           { return false }
func (*NullRowReader) RecordIDs(includeNulls bool, cb func(name string, value RecordID)) {}

// Implements IObject
type NullObject struct{ NullRowReader }

func NewNullObject() IObject { return &NullObject{} }

func (*NullObject) QName() QName                                    { return NullQName }
func (*NullObject) Elements(container string, cb func(el IElement)) {}
func (*NullObject) Containers(cb func(container string))            {}
func (no *NullObject) AsRecord() IRecord                            { return no }
func (no *NullObject) FieldNames(cb func(fieldName string))         {}
func (no *NullObject) Container() string                            { return "" }
func (no *NullObject) ID() RecordID                                 { return NullRecordID }
func (no *NullObject) Parent() RecordID                             { return NullRecordID }

// *********************************************************************************************************
//
//	ContainerOccursType
//

func (o ContainerOccursType) String() string {
	switch o {
	case ContainerOccurs_Unbounded:
		return ContainerOccurs_UnboundedStr
	default:
		const base = 10
		return strconv.FormatUint(uint64(o), base)
	}
}

func (o ContainerOccursType) MarshalJSON() ([]byte, error) {
	s := o.String()
	switch o {
	case ContainerOccurs_Unbounded:
		s = strconv.Quote(s)
	}
	return []byte(s), nil
}

func (o *ContainerOccursType) UnmarshalJSON(data []byte) (err error) {
	switch string(data) {
	case strconv.Quote(ContainerOccurs_UnboundedStr):
		*o = ContainerOccurs_Unbounded
		return nil
	default:
		var i uint64
		const base, wordBits = 10, 16
		i, err = strconv.ParseUint(string(data), base, wordBits)
		if err == nil {
			*o = ContainerOccursType(i)
		}
		return err
	}
}

// *********************************************************************************************************
//
//	DataKindType
//

func (i DataKindType) MarshalText() ([]byte, error) {
	var s string
	if i < DataKind_FakeLast {
		s = i.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(i), base)
	}
	return []byte(s), nil
}

// *********************************************************************************************************
//
//	SchemaKindType
//

func (k SchemaKindType) MarshalText() ([]byte, error) {
	var s string
	if k < SchemaKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

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

func ValidatorMatchByQName(cudValidator CUDValidator, cudQName QName) bool {
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
