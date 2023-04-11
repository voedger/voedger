/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

func newField(name string, kind DataKind, required, verified bool) Field {
	return Field{name, kind, required, verified}
}

// Returns is field system
func (fld *Field) IsSys() bool {
	return IsSysField(fld.Name())
}

// Returns is field has fixed width data kind
func (fld *Field) IsFixedWidth() bool {
	return IsFixedWidthDataKind(fld.DataKind())
}

// ————————— istructs.IFieldDescr ——————————

func (fld *Field) Name() string { return fld.name }

func (fld *Field) DataKind() DataKind { return fld.kind }

func (fld *Field) Required() bool { return fld.required }

func (fld *Field) Verifiable() bool { return fld.verifiable }
