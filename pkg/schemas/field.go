/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

// Implements IField interface
type field struct {
	name       string
	kind       DataKind
	required   bool
	verifiable bool
}

func newField(name string, kind DataKind, required, verified bool) *field {
	return &field{name, kind, required, verified}
}

func (fld *field) IsSys() bool {
	return IsSysField(fld.Name())
}

func (fld *field) IsFixedWidth() bool {
	return IsFixedWidthDataKind(fld.DataKind())
}

func (fld *field) DataKind() DataKind { return fld.kind }

func (fld *field) Name() string { return fld.name }

func (fld *field) Required() bool { return fld.required }

func (fld *field) Verifiable() bool { return fld.verifiable }
