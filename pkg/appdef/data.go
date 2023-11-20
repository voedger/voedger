/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// #Implement:
//   - IData
// 	 - IDataBuilder
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
		case ConstraintKind_MaxLen:
			max := MaxFieldLength // string or []byte
			if dk == DataKind_raw {
				max = MaxRawFieldLength
			}
			if c.Value().(uint16) > max {
				panic(fmt.Errorf("constraint %v value %v exceeds maximum (%v): %w", c, c.Value(), max, ErrMaxFieldLengthExceeds))
			}
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

func (d *data) Constraints(f func(IConstraint)) {
	if len(d.constraints) > 0 {
		for i := ConstraintKind(1); i < ConstraintKind_Count; i++ {
			if c, ok := d.constraints[i]; ok {
				f(c)
			}
		}
	}
}

func (d *data) DataKind() DataKind {
	return d.dataKind
}

func (d *data) String() string {
	return fmt.Sprintf("%s-data «%v»", d.DataKind().TrimString(), d.QName())
}
