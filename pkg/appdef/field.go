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
	name        string
	data        IData
	required    bool
	verifiable  bool
	verify      map[VerificationKind]bool
	constraints map[ConstraintKind]IConstraint
}

func makeField(name string, data IData, required bool, comments ...string) field {
	f := field{
		comment:     makeComment(comments...),
		name:        name,
		data:        data,
		required:    required,
		verifiable:  false,
		constraints: DataConstraintsInherited(data),
	}
	return f
}

func newField(name string, data IData, required bool, comments ...string) *field {
	f := makeField(name, data, required, comments...)
	return &f
}

func (fld *field) Constraints(f func(IConstraint)) {
	for i := ConstraintKind(1); i < ConstraintKind_Count; i++ {
		if c, ok := fld.constraints[i]; ok {
			f(c)
		}
	}
}

func (fld *field) Data() IData { return fld.data }

func (fld *field) DataKind() DataKind { return fld.Data().DataKind() }

func (fld *field) IsFixedWidth() bool {
	return fld.DataKind().IsFixed()
}

func (fld *field) IsSys() bool {
	return IsSysField(fld.Name())
}

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
	app           *appDef
	emb           interface{}
	fields        map[string]interface{}
	fieldsOrdered []string
}

// Makes new fields instance
func makeFields(app *appDef, embeds interface{}) fields {
	ff := fields{
		app:           app,
		emb:           embeds,
		fields:        make(map[string]interface{}),
		fieldsOrdered: make([]string, 0)}
	return ff
}

func (ff *fields) AddBytesField(name string, required bool, constraints ...IConstraint) IFieldsBuilder {
	d := newAnonymousData(ff.app, DataKind_bytes, NullQName, constraints...)
	f := newField(name, d, required)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddField(name string, kind DataKind, required bool, comments ...string) IFieldsBuilder {
	d := ff.app.SysData(kind)
	if d == nil {
		panic(fmt.Errorf("%v: system data type for data kind «%s» is not exists: %w", ff.embeds(), kind.TrimString(), ErrInvalidTypeKind))
	}
	f := newField(name, d, required, comments...)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	d := ff.app.SysData(DataKind_RecordID)
	f := newRefField(name, d, required, ref...)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddStringField(name string, required bool, constraints ...IConstraint) IFieldsBuilder {
	d := newAnonymousData(ff.app, DataKind_string, NullQName, constraints...)
	f := newField(name, d, required)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddTypedField(name string, dataType QName, required bool, constraints ...IConstraint) IFieldsBuilder {
	d := ff.app.Data(dataType)
	if d == nil {
		panic(fmt.Errorf("%v: data type «%v» not found: %w", ff.embeds(), dataType, ErrNameNotFound))
	}
	if len(constraints) > 0 {
		d = newAnonymousData(ff.app, d.DataKind(), dataType, constraints...)
	}
	f := newField(name, d, required)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) Field(name string) IField {
	if ff, ok := ff.fields[name]; ok {
		return ff.(IField)
	}
	return nil
}

func (ff *fields) FieldCount() int {
	return len(ff.fieldsOrdered)
}

func (ff *fields) Fields(cb func(IField)) {
	for _, n := range ff.fieldsOrdered {
		cb(ff.Field(n))
	}
}

func (ff *fields) RefField(name string) (rf IRefField) {
	if fld := ff.Field(name); fld != nil {
		if fld.DataKind() == DataKind_RecordID {
			if fld, ok := fld.(IRefField); ok {
				rf = fld
			}
		}
	}
	return rf
}

func (ff *fields) RefFields(cb func(IRefField)) {
	ff.Fields(func(fld IField) {
		if fld.DataKind() == DataKind_RecordID {
			if rf, ok := fld.(IRefField); ok {
				cb(rf)
			}
		}
	})
}

func (ff *fields) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	fld := ff.fields[name]
	if fld == nil {
		panic(fmt.Errorf("%v: field «%s» not found: %w", ff.embeds(), name, ErrNameNotFound))
	}
	fld.(ICommentBuilder).SetComment(comment...)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
	fld := ff.fields[name]
	if fld == nil {
		panic(fmt.Errorf("%v: field «%s» not found: %w", ff.embeds(), name, ErrNameNotFound))
	}
	vf := fld.(interface{ setVerify(k ...VerificationKind) })
	vf.setVerify(vk...)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) UserFields(cb func(IField)) {
	ff.Fields(func(fld IField) {
		if !fld.IsSys() {
			cb(fld)
		}
	})
}

func (ff *fields) UserFieldCount() int {
	cnt := 0
	ff.UserFields(func(IField) { cnt++ })
	return cnt
}

// Appends specified field.
//
// # Panics:
//   - if field name is empty,
//   - if field with specified name is already exists
//   - if user field name is invalid
//   - if user field data kind is not allowed by structured type kind
func (ff *fields) appendField(name string, fld interface{}) {
	if name == NullName {
		panic(fmt.Errorf("%v: empty field name: %w", ff.embeds(), ErrNameMissed))
	}
	if ff.Field(name) != nil {
		panic(fmt.Errorf("%v: field «%s» is already exists: %w", ff.embeds(), name, ErrNameUniqueViolation))
	}
	if len(ff.fields) >= MaxTypeFieldCount {
		panic(fmt.Errorf("%v: maximum field count (%d) exceeds: %w", ff.embeds(), MaxTypeFieldCount, ErrTooManyFields))
	}

	if !IsSysField(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("%v: field name «%v» is invalid: %w", ff.embeds(), name, err))
		}
		dk := fld.(IField).DataKind()
		tk := ff.embeds().Kind()
		if !tk.DataKindAvailable(dk) {
			panic(fmt.Errorf("%v: does not support %s-data fields: %w", ff.embeds(), dk.TrimString(), ErrInvalidDataKind))
		}
	}

	ff.fields[name] = fld
	ff.fieldsOrdered = append(ff.fieldsOrdered, name)
}

// Returns type that embeds fields
func (ff *fields) embeds() IType {
	return ff.emb.(IType)
}

// Makes system fields. Called after making structures fields
func (ff *fields) makeSysFields(k TypeKind) {
	if k.HasSystemField(SystemField_QName) {
		ff.AddField(SystemField_QName, DataKind_QName, true)
	}

	if k.HasSystemField(SystemField_ID) {
		ff.AddField(SystemField_ID, DataKind_RecordID, true)
	}

	if k.HasSystemField(SystemField_ParentID) {
		ff.AddField(SystemField_ParentID, DataKind_RecordID, true)
	}

	if k.HasSystemField(SystemField_Container) {
		ff.AddField(SystemField_Container, DataKind_string, true)
	}

	if k.HasSystemField(SystemField_IsActive) {
		ff.AddField(SystemField_IsActive, DataKind_bool, false)
	}
}

// # Implements:
//   - IRefField
type refField struct {
	field
	refs []QName
}

func newRefField(name string, data IData, required bool, ref ...QName) *refField {
	f := &refField{
		field: makeField(name, data, required),
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
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to not a record type %v: %w", t, n, refType, ErrInvalidTypeKind))
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
