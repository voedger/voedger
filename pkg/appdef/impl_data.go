/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// # Implements:
//   - IData
type data struct {
	typ
	dataKind    DataKind
	ancestor    IData
	constraints map[ConstraintKind]IConstraint
}

// Creates and returns new data type.
func newData(app *appDef, ws *workspace, name QName, kind DataKind, anc QName) *data {
	var ancestor IData
	if anc == NullQName {
		ancestor = SysData(app.Type, kind)
		if ancestor == nil {
			panic(ErrNotFound("system data type for data kind «%v»", kind.TrimString()))
		}
	} else {
		ancestor = Data(app.Type, anc)
		if ancestor == nil {
			panic(ErrTypeNotFound(anc))
		}
		if (kind != DataKind_null) && (ancestor.DataKind() != kind) {
			panic(ErrInvalid("ancestor «%v» has wrong data kind, expected %v", anc, kind.TrimString()))
		}
	}
	d := &data{
		typ:         makeType(app, ws, name, TypeKind_Data),
		dataKind:    ancestor.DataKind(),
		ancestor:    ancestor,
		constraints: make(map[ConstraintKind]IConstraint),
	}
	return d
}

// Creates and returns new anonymous data type with specified constraints.
func newAnonymousData(app *appDef, ws *workspace, kind DataKind, anc QName, constraints ...IConstraint) *data {
	d := newData(app, ws, NullQName, kind, anc)
	d.addConstraints(constraints...)
	return d
}

func (d *data) Ancestor() IData {
	return d.ancestor
}

func (d *data) Constraints(withInherited bool) map[ConstraintKind]IConstraint {
	if !withInherited {
		return d.constraints
	}

	cc := make(map[ConstraintKind]IConstraint)
	for a := d; a != nil; {
		for k, c := range a.constraints {
			if _, ok := cc[k]; !ok {
				cc[k] = c
			}
		}
		if a.ancestor == nil {
			break
		}
		a = a.ancestor.(*data)
	}
	return cc
}

func (d *data) DataKind() DataKind {
	return d.dataKind
}

func (d *data) String() string {
	return fmt.Sprintf("%s-data «%v»", d.DataKind().TrimString(), d.QName())
}

func (d *data) addConstraints(cc ...IConstraint) {
	dk := d.DataKind()
	for _, c := range cc {
		ck := c.Kind()
		if ok := dk.IsCompatibleWithConstraint(ck); !ok {
			panic(ErrIncompatible("constraint %v with data type «%v»", c, d))
		}
		switch c.Kind() {
		case ConstraintKind_MinLen:
			// no errors expected
		case ConstraintKind_MaxLen:
			// no errors expected
		case ConstraintKind_Enum:
			ok := false
			switch dk {
			case DataKind_int32:
				_, ok = c.Value().([]int32)
			case DataKind_int64:
				_, ok = c.Value().([]int64)
			case DataKind_float32:
				_, ok = c.Value().([]float32)
			case DataKind_float64:
				_, ok = c.Value().([]float64)
			case DataKind_string:
				_, ok = c.Value().([]string)
			}
			if !ok {
				panic(ErrIncompatible("values type «%T» with data type «%v»", c.Value(), d))
			}
		}
		d.constraints[ck] = c
	}
}

// # Implements:
//   - IDataBuilder
type dataBuilder struct {
	typeBuilder
	*data
}

func newDataBuilder(data *data) *dataBuilder {
	return &dataBuilder{
		typeBuilder: makeTypeBuilder(&data.typ),
		data:        data,
	}
}

func (db *dataBuilder) AddConstraints(cc ...IConstraint) IDataBuilder {
	db.data.addConstraints(cc...)
	return db
}

// Returns name of system data type by data kind.
//
// Returns NullQName if data kind is out of bounds.
func SysDataName(k DataKind) QName {
	if (k > DataKind_null) && (k < DataKind_FakeLast) {
		return NewQName(SysPackage, k.TrimString())
	}
	return NullQName
}

var (
	// System data type names
	SysData_int32    QName = SysDataName(DataKind_int32)
	SysData_int64    QName = SysDataName(DataKind_int64)
	SysData_float32  QName = SysDataName(DataKind_float32)
	SysData_float64  QName = SysDataName(DataKind_float64)
	SysData_bytes    QName = SysDataName(DataKind_bytes)
	SysData_String   QName = SysDataName(DataKind_string)
	SysData_QName    QName = SysDataName(DataKind_QName)
	SysData_bool     QName = SysDataName(DataKind_bool)
	SysData_RecordID QName = SysDataName(DataKind_RecordID)
)

// Creates and returns new system type by data kind.
func newSysData(app *appDef, ws *workspace, kind DataKind) *data {
	d := &data{
		typ:      makeType(app, ws, SysDataName(kind), TypeKind_Data),
		dataKind: kind,
	}
	ws.appendType(d)
	return d
}

// Returns is fixed width data kind
func (k DataKind) IsFixed() bool {
	switch k {
	case
		DataKind_int32,
		DataKind_int64,
		DataKind_float32,
		DataKind_float64,
		DataKind_QName,
		DataKind_bool,
		DataKind_RecordID:
		return true
	}
	return false
}

// # Implements:
//   - IDataConstraint
type dataConstraint struct {
	comment
	kind  ConstraintKind
	value any
}

// Creates and returns new data constraint.
func newDataConstraint(k ConstraintKind, v any, c ...string) IConstraint {
	return &dataConstraint{
		comment: makeComment(c...),
		kind:    k,
		value:   v,
	}
}

func (c dataConstraint) Kind() ConstraintKind {
	return c.kind
}

func (c dataConstraint) Value() any {
	return c.value
}

func (c dataConstraint) String() (s string) {
	const (
		maxLen   = 64
		ellipsis = `…`
	)

	switch c.kind {
	case ConstraintKind_Pattern:
		s = fmt.Sprintf("%s: `%v`", c.kind.TrimString(), c.value)
	case ConstraintKind_Enum:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	default:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	}
	if len(s) > maxLen {
		s = s[:maxLen-1] + ellipsis
	}
	return s
}

func (k ConstraintKind) MarshalText() ([]byte, error) {
	var s string
	if k < ConstraintKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an DataConstraintKind in human-readable form, without "DataConstraintKind_" prefix,
// suitable for debugging or error messages
func (k ConstraintKind) TrimString() string {
	const pref = "ConstraintKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Returns is data kind supports specified constraint kind.
//
// # Bytes data supports:
//   - ConstraintKind_MinLen
//   - ConstraintKind_MaxLen
//   - ConstraintKind_Pattern
//
// # String data supports:
//   - ConstraintKind_MinLen
//   - ConstraintKind_MaxLen
//   - ConstraintKind_Pattern
//   - ConstraintKind_Enum
//
// # Numeric data supports:
//   - ConstraintKind_MinIncl
//   - ConstraintKind_MinExcl
//   - ConstraintKind_MaxIncl
//   - ConstraintKind_MaxExcl
//   - ConstraintKind_Enum
func (k DataKind) IsCompatibleWithConstraint(c ConstraintKind) bool {
	switch k {
	case DataKind_bytes:
		switch c {
		case
			ConstraintKind_MinLen,
			ConstraintKind_MaxLen,
			ConstraintKind_Pattern:
			return true
		}
	case DataKind_string:
		switch c {
		case
			ConstraintKind_MinLen,
			ConstraintKind_MaxLen,
			ConstraintKind_Pattern,
			ConstraintKind_Enum:
			return true
		}
	case DataKind_int32, DataKind_int64, DataKind_float32, DataKind_float64:
		switch c {
		case
			ConstraintKind_MinIncl,
			ConstraintKind_MinExcl,
			ConstraintKind_MaxIncl,
			ConstraintKind_MaxExcl,
			ConstraintKind_Enum:
			return true
		}
	}
	return false
}

func (k DataKind) MarshalText() ([]byte, error) {
	var s string
	if k < DataKind_FakeLast {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an DataKind in human-readable form, without "DataKind_" prefix,
// suitable for debugging or error messages
func (k DataKind) TrimString() string {
	const pref = "DataKind_"
	return strings.TrimPrefix(k.String(), pref)
}
