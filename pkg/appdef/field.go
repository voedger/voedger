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
	comment
	name       string
	kind       DataKind
	required   bool
	verifiable bool
	verify     map[VerificationKind]bool
}

func makeField(name string, kind DataKind, required bool, comments ...string) field {
	f := field{comment{}, name, kind, required, false, make(map[VerificationKind]bool)}
	f.SetComment(comments...)
	return f
}

func newField(name string, kind DataKind, required bool, comments ...string) *field {
	f := makeField(name, kind, required, comments...)
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

func (fld *field) setVerify(k ...VerificationKind) {
	fld.verify = make(map[VerificationKind]bool)
	for _, kind := range k {
		fld.verify[kind] = true
	}
	fld.verifiable = len(fld.verify) > 0
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
	par           interface{}
	fields        map[string]interface{}
	fieldsOrdered []string
}

// Makes new fields instance
func makeFields(parent interface{}) fields {
	f := fields{parent, make(map[string]interface{}), make([]string, 0)}
	return f
}

func (f *fields) AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	f.checkAddField(name, DataKind_bytes)
	f.appendField(name, newCharsField(name, DataKind_bytes, required, restricts...))
	return f.par.(IFieldsBuilder)
}

func (f *fields) AddField(name string, kind DataKind, required bool, comments ...string) IFieldsBuilder {
	f.checkAddField(name, kind)
	fld := newField(name, kind, required, comments...)
	f.appendField(name, fld)
	return f.par.(IFieldsBuilder)
}

func (f *fields) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	f.checkAddField(name, DataKind_RecordID)
	f.appendField(name, newRefField(name, required, ref...))
	return f.par.(IFieldsBuilder)
}

func (f *fields) AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	f.checkAddField(name, DataKind_string)
	f.appendField(name, newCharsField(name, DataKind_string, required, restricts...))
	return f.par.(IFieldsBuilder)
}

func (f *fields) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	f.checkAddField(name, kind)
	if len(vk) == 0 {
		panic(fmt.Errorf("%v: missed verification kind for field «%s»: %w", f.parent(), name, ErrVerificationKindMissed))
	}
	f.appendField(name, newField(name, kind, required))
	f.SetFieldVerify(name, vk...)
	return f.par.(IFieldsBuilder)
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

func (f *fields) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	fld := f.fields[name]
	if fld == nil {
		panic(fmt.Errorf("%v: field «%s» not found: %w", f.parent(), name, ErrNameNotFound))
	}
	fld.(ICommentBuilder).SetComment(comment...)
	return f.par.(IFieldsBuilder)
}

func (f *fields) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
	fld := f.fields[name]
	if fld == nil {
		panic(fmt.Errorf("%v: field «%s» not found: %w", f.parent(), name, ErrNameNotFound))
	}
	vf := fld.(interface{ setVerify(k ...VerificationKind) })
	vf.setVerify(vk...)
	return f.par.(IFieldsBuilder)
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

// Panics if invalid add field argument value
func (f *fields) checkAddField(name string, kind DataKind) {
	if name == NullName {
		panic(fmt.Errorf("%v: empty field name: %w", f.parent(), ErrNameMissed))
	}
	if f.Field(name) != nil {
		panic(fmt.Errorf("%v: field «%s» is already exists: %w", f.parent(), name, ErrNameUniqueViolation))
	}

	if IsSysField(name) {
		return
	}

	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: field name «%v» is invalid: %w", f.parent(), name, err))
	}
	if k := f.parent().Kind(); !k.DataKindAvailable(kind) {
		panic(fmt.Errorf("%v: type kind «%s» does not support fields kind «%s»: %w", f.parent(), k.TrimString(), kind.TrimString(), ErrInvalidDataKind))
	}
	if len(f.fields) >= MaxTypeFieldCount {
		panic(fmt.Errorf("%v: maximum field count (%d) exceeds: %w", f.parent(), MaxTypeFieldCount, ErrTooManyFields))
	}
}

func (f *fields) parent() IType {
	return f.par.(IType)
}

// Makes system fields. Called after making structures fields
func (f *fields) makeSysFields(k TypeKind) {
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
		field: makeField(name, DataKind_RecordID, required),
		refs:  append([]QName{}, ref...),
	}
	return f
}

func (f refField) Ref(n QName) bool {
	l := len(f.refs)
	if l == 0 {
		return true // any ref available
	}
	for i := 0; i < l; i++ {
		if f.refs[i] == n {
			return true
		}
	}
	return false
}

func (f refField) Refs() []QName { return f.refs }

// Chars (string or bytes) field.
//
// # Implements:
//   - IStringField
//   - IBytesField
type charsField struct {
	field
	restricts *fieldRestricts
}

func newCharsField(name string, kind DataKind, required bool, restricts ...IFieldRestrict) *charsField {
	f := &charsField{field: makeField(name, kind, required)}
	f.restricts = newFieldRestricts(&f.field, restricts...)
	return f
}

func (f charsField) Restricts() IStringFieldRestricts {
	return f.restricts
}

// Validates specified fields.
//
// # Validation:
//   - every RefField must refer to known types,
//   - every referenced by RefField type must be record type
func validateTypeFields(t IType) (err error) {
	if fld, ok := t.(IFields); ok {
		// resolve reference types
		fld.RefFields(func(rf IRefField) {
			for _, n := range rf.Refs() {
				refType := t.App().TypeByName(n)
				if refType == nil {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to unknown type «%v»: %w", t, rf.Name(), n, ErrNameNotFound))
					continue
				}
				if _, ok := refType.(IRecord); !ok {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to non-record type %v: %w", t, n, refType, ErrInvalidTypeKind))
					continue
				}
			}
		})
	}
	return err
}

// NullFields is used for return then IFields is not supported
var NullFields = new(nullFields)

type nullFields struct{}

func (f *nullFields) Field(name string) IField       { return nil }
func (f *nullFields) FieldCount() int                { return 0 }
func (f *nullFields) Fields(func(IField))            {}
func (f *nullFields) RefField(name string) IRefField { return nil }
func (f *nullFields) RefFields(func(IRefField))      {}
func (f *nullFields) RefFieldCount() int             { return 0 }
func (f *nullFields) UserFields(func(IField))        {}
func (f *nullFields) UserFieldCount() int            { return 0 }
