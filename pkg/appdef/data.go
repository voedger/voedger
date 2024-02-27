/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strconv"
	"strings"
)

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
