/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package appdef

import (
	"errors"
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

func makeField(name string, kind DataKind, required, verified bool, vk ...VerificationKind) field {
	f := field{name, kind, required, verified, make(map[VerificationKind]bool)}
	if verified {
		for _, kind := range vk {
			f.verify[kind] = true
		}
	}
	return f
}

func newField(name string, kind DataKind, required, verified bool, vk ...VerificationKind) *field {
	f := makeField(name, kind, required, verified, vk...)
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
//   - IFields
//   - IFieldsBuilder
type fields struct {
	parent        interface{}
	fields        map[string]interface{}
	fieldsOrdered []string
}

func makeFields(def interface{}) fields {
	f := fields{def, make(map[string]interface{}), make([]string, 0)}
	f.makeSysFields()
	return f
}

func (f *fields) AddField(name string, kind DataKind, required bool) IFieldsBuilder {
	if err := f.checkAddField(name, kind); err != nil {
		panic(err)
	}
	f.appendField(name, newField(name, kind, required, false))
	return f.parent.(IFieldsBuilder)
}

func (f *fields) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	if err := f.checkAddField(name, DataKind_RecordID); err != nil {
		panic(err)
	}
	f.appendField(name, newRefField(name, required, ref...))
	return f.parent.(IFieldsBuilder)
}

func (f *fields) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	if err := f.checkAddField(name, kind); err != nil {
		panic(err)
	}
	if len(vk) == 0 {
		panic(fmt.Errorf("%v: missed verification kind for field «%s»: %w", f.parentDef().QName(), name, ErrVerificationKindMissed))
	}
	f.appendField(name, newField(name, kind, required, true, vk...))
	return f.parent.(IFieldsBuilder)
}

func (f *fields) Field(name string) IField {
	if f, ok := f.fields[name]; ok {
		return f.(IField)
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

func (f *fields) RefField(name string) (rf IRefField) {
	if fld := f.Field(name); fld != nil {
		if fld.DataKind() == DataKind_RecordID {
			if fld, ok := fld.(IRefField); ok {
				rf = fld
			}
		}
	}
	return rf
}

func (f *fields) RefFields(cb func(IRefField)) {
	f.Fields(func(fld IField) {
		if fld.DataKind() == DataKind_RecordID {
			if rf, ok := fld.(IRefField); ok {
				cb(rf)
			}
		}
	})
}

func (f *fields) RefFieldCount() int {
	cnt := 0
	f.Fields(func(fld IField) {
		if fld.DataKind() == DataKind_RecordID {
			if _, ok := fld.(IRefField); ok {
				cnt++
			}
		}
	})
	return cnt
}

func (f *fields) UserFields(cb func(IField)) {
	f.Fields(func(fld IField) {
		if !fld.IsSys() {
			cb(fld)
		}
	})
}

func (f *fields) UserFieldCount() int {
	cnt := 0
	f.Fields(func(fld IField) {
		if !fld.IsSys() {
			cnt++
		}
	})
	return cnt
}

func (f *fields) appendField(name string, fld interface{}) {
	f.fields[name] = fld
	f.fieldsOrdered = append(f.fieldsOrdered, name)
}

func (f *fields) checkAddField(name string, kind DataKind) error {
	if name == NullName {
		return fmt.Errorf("%v: empty field name: %w", f.parentDef().QName(), ErrNameMissed)
	}
	if !IsSysField(name) {
		if ok, err := ValidIdent(name); !ok {
			return fmt.Errorf("%v: field name «%v» is invalid: %w", f.parentDef().QName(), name, err)
		}
	}
	if f.Field(name) != nil {
		return fmt.Errorf("%v: field «%s» is already exists: %w", f.parentDef().QName(), name, ErrNameUniqueViolation)
	}

	if k := f.parentDef().Kind(); !k.DataKindAvailable(kind) {
		return fmt.Errorf("%v: definition kind «%s» does not support fields kind «%s»: %w", f.parentDef().QName(), k.TrimString(), kind.TrimString(), ErrInvalidDataKind)
	}

	if len(f.fields) >= MaxDefFieldCount {
		return fmt.Errorf("%v: maximum field count (%d) exceeds: %w", f.parentDef().QName(), MaxDefFieldCount, ErrTooManyFields)
	}

	return nil
}

func (f *fields) parentDef() IDef {
	return f.parent.(IDef)
}

func (f *fields) makeSysFields() {
	k := f.parentDef().Kind()

	if k.HasSystemField(SystemField_QName) {
		f.AddField(SystemField_QName, DataKind_QName, true)
	}

	if k.HasSystemField(SystemField_ID) {
		f.AddField(SystemField_ID, DataKind_RecordID, true)
	}

	if k.HasSystemField(SystemField_ParentID) {
		f.AddField(SystemField_ParentID, DataKind_RecordID, true)
	}

	if k.HasSystemField(SystemField_Container) {
		f.AddField(SystemField_Container, DataKind_string, true)
	}

	if k.HasSystemField(SystemField_IsActive) {
		f.AddField(SystemField_IsActive, DataKind_bool, false)
	}
}

// # Implements:
//   - IRefField
type refField struct {
	field
	refs []QName
}

func newRefField(name string, required bool, ref ...QName) *refField {
	f := &refField{
		field: makeField(name, DataKind_RecordID, required, false),
		refs:  append([]QName{}, ref...),
	}
	return f
}

func (f refField) Refs() []QName { return f.refs }

// Validates specified fields.
//
// # Validation:
//   - every RefField must refer to known definitions,
//   - every referenced by RefField definition must have «sys.ID» system field
func validateDefFields(def IDef) (err error) {
	if fld, ok := def.(IFields); ok {
		// resolve reference fields definitions
		fld.RefFields(func(rf IRefField) {
			for _, n := range rf.Refs() {
				refDef := def.App().DefByName(n)
				if refDef == nil {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to unknown definition «%v»: %w", def.QName(), rf.Name(), n, ErrNameNotFound))
					continue
				}
				if !refDef.Kind().HasSystemField(SystemField_ID) {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to non referable definition «%v» kind «%s» without «%s» field: %w", def.QName(), rf.Name(), n, refDef.Kind().TrimString(), SystemField_ID, ErrInvalidDefKind))
					continue
				}
			}
		})
	}
	return err
}
