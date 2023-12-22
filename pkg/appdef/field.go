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
		constraints: data.Constraints(true),
	}
	return f
}

func newField(name string, data IData, required bool, comments ...string) *field {
	f := makeField(name, data, required, comments...)
	return &f
}

func (fld *field) Constraints() map[ConstraintKind]IConstraint {
	return fld.constraints
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

func (fld field) String() string {
	return fmt.Sprintf("%s-field «%s»", fld.DataKind().TrimString(), fld.Name())
}

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
	fieldsOrdered []IField
	refFields     []IRefField
}

// Makes new fields instance
func makeFields(app *appDef, embeds interface{}) fields {
	ff := fields{
		app:           app,
		emb:           embeds,
		fields:        make(map[string]interface{}),
		fieldsOrdered: make([]IField, 0),
		refFields:     make([]IRefField, 0)}
	return ff
}

func (ff *fields) AddDataField(name string, data QName, required bool, constraints ...IConstraint) IFieldsBuilder {
	d := ff.app.Data(data)
	if d == nil {
		panic(fmt.Errorf("%v: data type «%v» not found: %w", ff.embeds(), data, ErrNameNotFound))
	}
	if len(constraints) > 0 {
		d = newAnonymousData(ff.app, d.DataKind(), data, constraints...)
	}
	f := newField(name, d, required)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddField(name string, kind DataKind, required bool, constraints ...IConstraint) IFieldsBuilder {
	d := ff.app.SysData(kind)
	if d == nil {
		panic(fmt.Errorf("%v: system data type for data kind «%s» is not exists: %w", ff.embeds(), kind.TrimString(), ErrInvalidTypeKind))
	}
	if len(constraints) > 0 {
		d = newAnonymousData(ff.app, d.DataKind(), d.QName(), constraints...)
	}
	f := newField(name, d, required)
	ff.appendField(name, f)
	return ff.emb.(IFieldsBuilder)
}

func (ff *fields) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	d := ff.app.SysData(DataKind_RecordID)
	f := newRefField(name, d, required, ref...)
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

func (ff *fields) Fields() []IField {
	return ff.fieldsOrdered
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

func (ff *fields) RefFields() []IRefField {
	return ff.refFields
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

func (ff *fields) UserFieldCount() int {
	cnt := 0
	for _, fld := range ff.fieldsOrdered {
		if !fld.IsSys() {
			cnt++
		}
	}
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
	ff.fieldsOrdered = append(ff.fieldsOrdered, fld.(IField))

	if rf, ok := fld.(IRefField); ok {
		ff.refFields = append(ff.refFields, rf)
	}
}

// Returns type that embeds fields
func (ff *fields) embeds() IType {
	return ff.emb.(IType)
}

// Makes system fields. Called after making structures fields
func (ff *fields) makeSysFields(k TypeKind) {
	if exists, required := k.HasSystemField(SystemField_QName); exists {
		ff.AddField(SystemField_QName, DataKind_QName, required)
	}

	if exists, required := k.HasSystemField(SystemField_ID); exists {
		ff.AddField(SystemField_ID, DataKind_RecordID, required)
	}

	if exists, required := k.HasSystemField(SystemField_ParentID); exists {
		ff.AddField(SystemField_ParentID, DataKind_RecordID, required)
	}

	if exists, required := k.HasSystemField(SystemField_Container); exists {
		ff.AddField(SystemField_Container, DataKind_string, required)
	}

	if exists, required := k.HasSystemField(SystemField_IsActive); exists {
		ff.AddField(SystemField_IsActive, DataKind_bool, required)
	}
}

// # Implements:
//   - IRefField
type refField struct {
	field
	refs QNames
}

func newRefField(name string, data IData, required bool, ref ...QName) *refField {
	f := &refField{
		field: makeField(name, data, required),
		refs:  QNames{},
	}
	f.refs.Add(ref...)
	return f
}

func (f refField) Ref(n QName) bool {
	l := len(f.refs)
	if l == 0 {
		return true // any ref available
	}
	return f.refs.Contains(n)
}

func (f refField) Refs() QNames { return f.refs }

// Validates specified fields.
//
// # Validation:
//   - every RefField must refer to known types,
//   - every referenced by RefField type must be record type
func validateTypeFields(t IType) (err error) {
	if ff, ok := t.(IFields); ok {
		// resolve reference types
		for _, rf := range ff.RefFields() {
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
		}
	}
	return err
}

// NullFields is used for return then IFields is not supported
var NullFields = new(nullFields)

type nullFields struct{}

func (f *nullFields) Field(name string) IField       { return nil }
func (f *nullFields) FieldCount() int                { return 0 }
func (f *nullFields) Fields() []IField               { return []IField{} }
func (f *nullFields) RefField(name string) IRefField { return nil }
func (f *nullFields) RefFields() []IRefField         { return []IRefField{} }
func (f *nullFields) UserFieldCount() int            { return 0 }
