/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package appdef

import (
	"fmt"
	"strings"
)

const (
	SystemField_ID        = SystemPackagePrefix + "ID"
	SystemField_ParentID  = SystemPackagePrefix + "ParentID"
	SystemField_IsActive  = SystemPackagePrefix + "IsActive"
	SystemField_Container = SystemPackagePrefix + "Container"
	SystemField_QName     = SystemPackagePrefix + "QName"
)

// # Implements:
//   - IField
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

// # Implements:
//   - IWithFields
//   - IFieldsBuilder
type fields struct {
	def           *def
	fields        map[string]*field
	fieldsOrdered []string
}

func makeFields(def *def) fields {
	f := fields{def, make(map[string]*field), make([]string, 0)}
	if def.Kind().FieldsAllowed() {
		f.makeSysFields()
	}
	return f
}

func (f *fields) AddField(name string, kind DataKind, required bool) IFieldsBuilder {
	f.addField(name, kind, required, false)
	return f
}

func (f *fields) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	f.addField(name, kind, required, true, vk...)
	return f
}

func (f *fields) Field(name string) IField {
	if f, ok := f.fields[name]; ok {
		return f
	}
	return nil
}

func (f *fields) FieldCount() int {
	return len(f.fieldsOrdered)
}

func (f *fields) Fields(cb func(IField)) {
	for _, n := range f.fieldsOrdered {
		cb(f.Field(n))
	}
}

func (f *fields) addField(name string, kind DataKind, required, verified bool, vk ...VerificationKind) {
	if name == NullName {
		panic(fmt.Errorf("%v: empty field name: %w", f.def.QName(), ErrNameMissed))
	}
	if !IsSysField(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("%v: field name «%v» is invalid: %w", f.def.QName(), name, err))
		}
	}
	if f.Field(name) != nil {
		if IsSysField(name) {
			return
		}
		panic(fmt.Errorf("%v: definition field «%v» is already exists: %w", f.def.QName(), name, ErrNameUniqueViolation))
	}
	if !f.def.Kind().FieldsAllowed() {
		panic(fmt.Errorf("%v: definition kind «%v» does not allow fields: %w", f.def.QName(), f.def.Kind(), ErrInvalidDefKind))
	}
	if !f.def.Kind().DataKindAvailable(kind) {
		panic(fmt.Errorf("%v: definition kind «%v» does not support fields kind «%v»: %w", f.def.QName(), f.def.Kind(), kind, ErrInvalidDataKind))
	}

	if verified && (len(vk) == 0) {
		panic(fmt.Errorf("%v: missed verification kind for field «%v»: %w", f.def.QName(), name, ErrVerificationKindMissed))
	}

	if len(f.fields) >= MaxDefFieldCount {
		panic(fmt.Errorf("%v: maximum field count (%d) exceeds: %w", f.def.QName(), MaxDefFieldCount, ErrTooManyFields))
	}

	fld := newField(name, kind, required, verified, vk...)
	f.fields[name] = fld
	f.fieldsOrdered = append(f.fieldsOrdered, name)

	f.def.changed()
}

func (f *fields) makeSysFields() {
	if f.def.Kind().HasSystemField(SystemField_QName) {
		f.AddField(SystemField_QName, DataKind_QName, true)
	}

	if f.def.Kind().HasSystemField(SystemField_ID) {
		f.AddField(SystemField_ID, DataKind_RecordID, true)
	}

	if f.def.Kind().HasSystemField(SystemField_ParentID) {
		f.AddField(SystemField_ParentID, DataKind_RecordID, true)
	}

	if f.def.Kind().HasSystemField(SystemField_Container) {
		f.AddField(SystemField_Container, DataKind_string, true)
	}

	if f.def.Kind().HasSystemField(SystemField_IsActive) {
		f.AddField(SystemField_IsActive, DataKind_bool, false)
	}
}
