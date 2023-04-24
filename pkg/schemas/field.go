/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package schemas

import (
	"strings"
)

const (
	SystemField_ID        = SystemPackagePrefix + "ID"
	SystemField_ParentID  = SystemPackagePrefix + "ParentID"
	SystemField_IsActive  = SystemPackagePrefix + "IsActive"
	SystemField_Container = SystemPackagePrefix + "Container"
	SystemField_QName     = SystemPackagePrefix + "QName"
)

// Implements IField interface
type field struct {
	name       string
	kind       DataKind
	required   bool
	verifiable bool
	verify     map[VerificationKind]bool
}

func newField(name string, kind DataKind, required, verified bool, vk ...VerificationKind) *field {
	f := field{name, kind, required, verified, make(map[VerificationKind]bool)}
	if verified {
		for _, kind := range vk {
			f.verify[kind] = true
		}
	}
	return &f
}

func (fld *field) IsSys() bool {
	return IsSysField(fld.Name())
}

func (fld *field) IsFixedWidth() bool {
	return fld.DataKind().IsFixed()
}

func (fld *field) DataKind() DataKind { return fld.kind }

func (fld *field) Name() string { return fld.name }

func (fld *field) Required() bool { return fld.required }

func (fld *field) Verifiable() bool { return fld.verifiable }

func (fld *field) VerificationKind(vk VerificationKind) bool {
	return fld.verifiable && fld.verify[vk]
}

// Returns is field system
func IsSysField(n string) bool {
	return strings.HasPrefix(n, SystemPackagePrefix) && // fast check
		// then more accuracy
		((n == SystemField_QName) ||
			(n == SystemField_ID) ||
			(n == SystemField_ParentID) ||
			(n == SystemField_Container) ||
			(n == SystemField_IsActive))
}
