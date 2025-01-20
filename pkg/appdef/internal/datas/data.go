/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package datas

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IData
type Data struct {
	types.Typ
	dataKind    appdef.DataKind
	ancestor    appdef.IData
	constraints map[appdef.ConstraintKind]appdef.IConstraint
}

// Creates and returns new data type.
func NewData(ws appdef.IWorkspace, name appdef.QName, kind appdef.DataKind, anc appdef.QName) *Data {
	var ancestor appdef.IData
	if anc == appdef.NullQName {
		ancestor = appdef.SysData(ws.Type, kind)
		if ancestor == nil {
			panic(appdef.ErrNotFound("system data type for data kind «%v»", kind.TrimString()))
		}
	} else {
		ancestor = appdef.Data(ws.Type, anc)
		if ancestor == nil {
			panic(appdef.ErrTypeNotFound(anc))
		}
		if (kind != appdef.DataKind_null) && (ancestor.DataKind() != kind) {
			panic(appdef.ErrInvalid("ancestor «%v» has wrong data kind, expected %v", anc, kind.TrimString()))
		}
	}
	d := &Data{
		Typ:         types.MakeType(ws.App(), ws, name, appdef.TypeKind_Data),
		dataKind:    ancestor.DataKind(),
		ancestor:    ancestor,
		constraints: make(map[appdef.ConstraintKind]appdef.IConstraint),
	}

	if name != appdef.NullQName {
		types.Propagate(d)
	}

	return d
}

// Creates and returns new anonymous data type with specified constraints.
func NewAnonymousData(ws appdef.IWorkspace, kind appdef.DataKind, anc appdef.QName, constraints ...appdef.IConstraint) *Data {
	d := NewData(ws, appdef.NullQName, kind, anc)
	d.addConstraints(constraints...)
	return d
}

func (d Data) Ancestor() appdef.IData { return d.ancestor }

func (d Data) Constraints(withInherited bool) map[appdef.ConstraintKind]appdef.IConstraint {
	if !withInherited || (d.ancestor == nil) || d.ancestor.IsSystem() {
		return d.constraints
	}

	cc := make(map[appdef.ConstraintKind]appdef.IConstraint)
	for a := &d; !a.IsSystem(); a = a.ancestor.(*Data) {
		for k, c := range a.constraints {
			if _, ok := cc[k]; !ok {
				cc[k] = c
			}
		}
	}
	return cc
}

func (d Data) DataKind() appdef.DataKind { return d.dataKind }

func (d Data) String() string {
	s := fmt.Sprintf("%s-data", d.DataKind().TrimString())
	if n := d.QName(); n != appdef.NullQName {
		s += fmt.Sprintf(" «%v»", n)
	}
	return s
}

func (d *Data) addConstraints(cc ...appdef.IConstraint) {
	dk := d.DataKind()
	for _, c := range cc {
		ck := c.Kind()
		if ok := dk.IsCompatibleWithConstraint(ck); !ok {
			panic(appdef.ErrIncompatible("constraint %v with data type «%v»", c, d))
		}
		switch c.Kind() {
		case appdef.ConstraintKind_MinLen:
			// no errors expected
		case appdef.ConstraintKind_MaxLen:
			// no errors expected
		case appdef.ConstraintKind_Enum:
			ok := false
			switch dk {
			case appdef.DataKind_int32:
				_, ok = c.Value().([]int32)
			case appdef.DataKind_int64:
				_, ok = c.Value().([]int64)
			case appdef.DataKind_float32:
				_, ok = c.Value().([]float32)
			case appdef.DataKind_float64:
				_, ok = c.Value().([]float64)
			case appdef.DataKind_string:
				_, ok = c.Value().([]string)
			}
			if !ok {
				panic(appdef.ErrIncompatible("values type «%T» with data type «%v»", c.Value(), d))
			}
		}
		d.constraints[ck] = c
	}
}

// # Supports:
//   - appdef.IDataBuilder
type DataBuilder struct {
	types.TypeBuilder
	*Data
}

func NewDataBuilder(data *Data) *DataBuilder {
	return &DataBuilder{
		TypeBuilder: types.MakeTypeBuilder(&data.Typ),
		Data:        data,
	}
}

func (db *DataBuilder) AddConstraints(cc ...appdef.IConstraint) appdef.IDataBuilder {
	db.Data.addConstraints(cc...)
	return db
}

// Creates and returns new system type by data kind.
func NewSysData(ws appdef.IWorkspace, kind appdef.DataKind) *Data {
	d := &Data{
		Typ:      types.MakeType(ws.App(), ws, appdef.SysDataName(kind), appdef.TypeKind_Data),
		dataKind: kind,
	}
	types.Propagate(d)
	return d
}

// # Supports:
//   - appdef.IDataConstraint
type DataConstraint struct {
	comments.WithComments
	kind  appdef.ConstraintKind
	value any
}

// Creates and returns new data constraint.
func NewDataConstraint(k appdef.ConstraintKind, v any, c ...string) appdef.IConstraint {
	return &DataConstraint{
		WithComments: comments.MakeWithComments(c...),
		kind:         k,
		value:        v,
	}
}

func (c DataConstraint) Kind() appdef.ConstraintKind { return c.kind }

func (c DataConstraint) Value() any { return c.value }

func (c DataConstraint) String() (s string) {
	const (
		maxLen   = 64
		ellipsis = `…`
	)

	switch c.kind {
	case appdef.ConstraintKind_Pattern:
		s = fmt.Sprintf("%s: `%v`", c.kind.TrimString(), c.value)
	case appdef.ConstraintKind_Enum:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	default:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	}
	if len(s) > maxLen {
		s = s[:maxLen-1] + ellipsis
	}
	return s
}
