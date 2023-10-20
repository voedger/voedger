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
	constraints constraints
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
		constraints: makeConstraints(),
	}
	return d
}

// Creates and returns new anonymous data type with specified constraints.
func newAnonymousData(app *appDef, kind DataKind, anc QName, constraints ...IConstraint) *data {
	d := newData(app, NullQName, kind, anc)
	d.AddConstraints(constraints...)
	return d
}

func (d *data) AddConstraints(c ...IConstraint) IDataBuilder {
	d.constraints.set(d.DataKind(), c...)
	return d
}

func (d *data) Ancestor() IData {
	return d.ancestor
}

func (d *data) Constraints() IConstraints {
	return d.constraints
}

func (d *data) DataKind() DataKind {
	return d.dataKind
}

func (d *data) IsSystem() bool {
	return d.QName().Pkg() == SysPackage
}

func (d *data) String() string {
	return fmt.Sprintf("%s-data «%v»", d.DataKind().TrimString(), d.QName())
}
