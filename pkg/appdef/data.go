/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Return new minimum length constraint for string or bytes data types.
func MinLen(v uint16, c ...string) IConstraint {
	return newDataConstraint(ConstraintKind_MinLen, v, c...)
}

// Return new maximum length restriction for string or bytes data types.
//
// Using MaxLen(), you can both limit the maximum length by a smaller value,
// and increase it to MaxFieldLength (65535).
//
// # Panics:
//   - if value is zero
func MaxLen(v uint16, c ...string) IConstraint {
	if v == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MaxLen, v, c...)
}

// Return new pattern restriction for string or bytes data types.
//
// # Panics:
//   - if value is not valid regular expression
func Pattern(v string, c ...string) IConstraint {
	re, err := regexp.Compile(v)
	if err != nil {
		panic(err)
	}
	return newDataConstraint(ConstraintKind_Pattern, re, c...)
}

// Return new minimum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is +infinite
func MinIncl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("minimum inclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, 1) {
		panic(fmt.Errorf("minimum inclusive value is positive infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MinIncl, v, c...)
}

// Return new minimum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is +infinite
func MinExcl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("minimum exclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, 1) {
		panic(fmt.Errorf("minimum inclusive value is positive infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MinExcl, v, c...)
}

// Return new maximum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxIncl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("maximum inclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, -1) {
		panic(fmt.Errorf("maximum inclusive value is negative infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MaxIncl, v, c...)
}

// Return new maximum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxExcl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("maximum exclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, -1) {
		panic(fmt.Errorf("maximum exclusive value is negative infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MaxExcl, v, c...)
}

type enumerable interface {
	string | int32 | int64 | float32 | float64
}

// Return new enumeration constraint for char or numeric data types.
//
// Enumeration values must be one of the following types:
//   - string
//   - int32
//   - int64
//   - float32
//   - float64
//
// Passed values will be sorted and duplicates removed before placing
// into returning constraint.
//
// # Panics:
//   - if enumeration values list is empty
func Enum[T enumerable](v ...T) IConstraint {
	l := len(v)
	if l == 0 {
		panic(fmt.Errorf("enumeration values slice (%T) is empty: %w", v, ErrIncompatibleConstraints))
	}
	c := make([]T, 0, l)
	for i := 0; i < l; i++ {
		n := v[i]
		c = append(c, n)
	}
	slices.Sort(c)
	c = slices.Compact(c)
	return newDataConstraint(ConstraintKind_Enum, c)
}

// Creates and returns new constraint.
//
// # Panics:
//   - if kind is unknown,
//   - id value is not compatible with kind.
func NewConstraint(kind ConstraintKind, value any, c ...string) IConstraint {
	switch kind {
	case ConstraintKind_MinLen:
		return MinLen(value.(uint16), c...)
	case ConstraintKind_MaxLen:
		return MaxLen(value.(uint16), c...)
	case ConstraintKind_Pattern:
		return Pattern(value.(string), c...)
	case ConstraintKind_MinIncl:
		return MinIncl(value.(float64), c...)
	case ConstraintKind_MinExcl:
		return MinExcl(value.(float64), c...)
	case ConstraintKind_MaxIncl:
		return MaxIncl(value.(float64), c...)
	case ConstraintKind_MaxExcl:
		return MaxExcl(value.(float64), c...)
	case ConstraintKind_Enum:
		var enum IConstraint
		switch v := value.(type) {
		case []string:
			enum = Enum(v...)
		case []int32:
			enum = Enum(v...)
		case []int64:
			enum = Enum(v...)
		case []float32:
			enum = Enum(v...)
		case []float64:
			enum = Enum(v...)
		default:
			panic(fmt.Errorf("unsupported enumeration type: %T", value))
		}
		if len(c) > 0 {
			enum.(ICommentBuilder).SetComment(c...)
		}
		return enum
	}
	panic(fmt.Errorf("unknown constraint kind: %v", kind))
}

// #Implement:
//   - IData
//   - IDataBuilder
type data struct {
	typ
	dataKind    DataKind
	ancestor    IData
	constraints map[ConstraintKind]IConstraint
}

// Creates and returns new data type.
func newData(app *appDef, name QName, kind DataKind, anc QName) *data {
	var ancestor IData
	if anc == NullQName {
		ancestor = app.SysData(kind)
		if ancestor == nil {
			panic(fmt.Errorf("system data type for data kind «%s» is not exists: %w", kind.TrimString(), ErrInvalidTypeKind))
		}
	} else {
		ancestor = app.Data(anc)
		if ancestor == nil {
			panic(fmt.Errorf("ancestor data type «%v» not found: %w", anc, ErrNameNotFound))
		}
		if (kind != DataKind_null) && (ancestor.DataKind() != kind) {
			panic(fmt.Errorf("ancestor «%v» has wrong data type, %v expected: %w", anc, kind, ErrInvalidTypeKind))
		}
	}
	d := &data{
		typ:         makeType(app, name, TypeKind_Data),
		dataKind:    ancestor.DataKind(),
		ancestor:    ancestor,
		constraints: make(map[ConstraintKind]IConstraint),
	}
	return d
}

// Creates and returns new anonymous data type with specified constraints.
func newAnonymousData(app *appDef, kind DataKind, anc QName, constraints ...IConstraint) *data {
	d := newData(app, NullQName, kind, anc)
	d.AddConstraints(constraints...)
	return d
}

func (d *data) AddConstraints(cc ...IConstraint) IDataBuilder {
	dk := d.DataKind()
	for _, c := range cc {
		ck := c.Kind()
		if ok := dk.IsSupportedConstraint(ck); !ok {
			panic(fmt.Errorf("%v is not compatible with constraint %v: %w", d, c, ErrIncompatibleConstraints))
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
				panic(fmt.Errorf("constraint %v values type %T is not applicable to %v: %w", c, c.Value(), d, ErrIncompatibleConstraints))
			}
		}
		d.constraints[ck] = c
	}
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
func newSysData(app *appDef, kind DataKind) *data {
	d := &data{
		typ:      makeType(app, SysDataName(kind), TypeKind_Data),
		dataKind: kind,
	}
	app.appendType(d)
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
	if k < ConstraintKind_Count {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
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
func (k DataKind) IsSupportedConstraint(c ConstraintKind) bool {
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
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an DataKind in human-readable form, without "DataKind_" prefix,
// suitable for debugging or error messages
func (k DataKind) TrimString() string {
	const pref = "DataKind_"
	return strings.TrimPrefix(k.String(), pref)
}
