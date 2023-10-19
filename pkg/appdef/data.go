/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

type data struct {
	typ
	dataKind DataKind
	ancestor IData
}

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
	}
	d := &data{
		typ:      makeType(app, name, TypeKind_Data),
		dataKind: kind,
		ancestor: ancestor,
	}
	app.appendType(d)
	return d
}

func (d *data) Ancestor() IData {
	return d.ancestor
}

func (d *data) DataKind() DataKind {
	return d.dataKind
}

func (d *data) String() string {
	return fmt.Sprintf("%s-data «%v»", d.DataKind().TrimString(), d.QName())
}

func (d *data) System() bool {
	return d.QName().Pkg() == SysPackage
}
