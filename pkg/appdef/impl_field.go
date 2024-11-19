/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// # Implements:
//   - IField
type field struct {
	comment
	name        FieldName
	data        IData
	required    bool
	verifiable  bool
	verify      map[VerificationKind]bool
	constraints map[ConstraintKind]IConstraint
}

func makeField(name FieldName, data IData, required bool, comments ...string) field {
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

func newField(name FieldName, data IData, required bool, comments ...string) *field {
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

func (fld *field) Name() FieldName { return fld.name }

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
func IsSysField(n FieldName) bool {
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
type fields struct {
	app           *appDef
	ws            *workspace
	typeKind      TypeKind
	fields        map[FieldName]interface{}
	fieldsOrdered []IField
	refFields     []IRefField
	userFields    []IField
}

// Makes new fields instance
func makeFields(app *appDef, ws *workspace, typeKind TypeKind) fields {
	ff := fields{
		app:           app,
		ws:            ws,
		typeKind:      typeKind,
		fields:        make(map[FieldName]interface{}),
		fieldsOrdered: make([]IField, 0),
		refFields:     make([]IRefField, 0),
		userFields:    make([]IField, 0)}
	return ff
}

func (ff *fields) Field(name FieldName) IField {
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

func (ff *fields) RefField(name FieldName) (rf IRefField) {
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

func (ff *fields) UserFields() []IField {
	return ff.userFields
}

func (ff *fields) UserFieldCount() int {
	return len(ff.userFields)
}

func (ff *fields) addDataField(name FieldName, data QName, required bool, constraints ...IConstraint) {
	d := Data(ff.app.Type, data)
	if d == nil {
		panic(ErrTypeNotFound(data))
	}
	if len(constraints) > 0 {
		d = newAnonymousData(ff.app, ff.ws, d.DataKind(), data, constraints...)
	}
	f := newField(name, d, required)
	ff.appendField(name, f)
}

func (ff *fields) addField(name FieldName, kind DataKind, required bool, constraints ...IConstraint) {
	d := SysData(ff.app.Type, kind)
	if d == nil {
		panic(ErrNotFound("system data type for data kind «%s»", kind.TrimString()))
	}
	if len(constraints) > 0 {
		d = newAnonymousData(ff.app, ff.ws, d.DataKind(), d.QName(), constraints...)
	}
	f := newField(name, d, required)
	ff.appendField(name, f)
}

func (ff *fields) addRefField(name FieldName, required bool, ref ...QName) {
	d := SysData(ff.app.Type, DataKind_RecordID)
	f := newRefField(name, d, required, ref...)
	ff.appendField(name, f)
}

// Appends specified field.
//
// # Panics:
//   - if field name is empty,
//   - if field with specified name is already exists
//   - if user field name is invalid
//   - if user field data kind is not allowed by structured type kind
func (ff *fields) appendField(name FieldName, fld interface{}) {
	if name == NullName {
		panic(ErrMissed("field name"))
	}
	if ff.Field(name) != nil {
		panic(ErrAlreadyExists("field «%v»", name))
	}
	if len(ff.fields) >= MaxTypeFieldCount {
		panic(ErrTooMany("fields, maximum is %d", MaxTypeFieldCount))
	}

	f := fld.(IField)

	if !IsSysField(name) {
		if ok, err := ValidFieldName(name); !ok {
			panic(fmt.Errorf("field name «%v» is invalid: %w", name, err))
		}
		dk := f.DataKind()
		if (ff.typeKind != TypeKind_null) && !ff.typeKind.FieldKindAvailable(dk) {
			panic(ErrIncompatible("data kind «%s» with fields of «%v»", dk.TrimString(), ff.typeKind.TrimString()))
		}
	}

	ff.fields[name] = fld
	ff.fieldsOrdered = append(ff.fieldsOrdered, f)
	if !IsSysField(name) {
		ff.userFields = append(ff.userFields, f)
	}

	if rf, ok := fld.(IRefField); ok {
		ff.refFields = append(ff.refFields, rf)
	}
}

// Makes system fields. Called after making structures fields
func (ff *fields) makeSysFields() {
	if exists, required := ff.typeKind.HasSystemField(SystemField_QName); exists {
		ff.addField(SystemField_QName, DataKind_QName, required)
	}

	if exists, required := ff.typeKind.HasSystemField(SystemField_ID); exists {
		ff.addField(SystemField_ID, DataKind_RecordID, required)
	}

	if exists, required := ff.typeKind.HasSystemField(SystemField_ParentID); exists {
		ff.addField(SystemField_ParentID, DataKind_RecordID, required)
	}

	if exists, required := ff.typeKind.HasSystemField(SystemField_Container); exists {
		ff.addField(SystemField_Container, DataKind_string, required)
	}

	if exists, required := ff.typeKind.HasSystemField(SystemField_IsActive); exists {
		ff.addField(SystemField_IsActive, DataKind_bool, required)
	}
}

func (ff *fields) setFieldComment(name FieldName, comment ...string) {
	fld := ff.fields[name]
	if fld == nil {
		panic(ErrFieldNotFound(name))
	}
	if fld, ok := fld.(interface{ setComment(comment ...string) }); ok {
		fld.setComment(comment...)
	}
}

func (ff *fields) setFieldVerify(name FieldName, vk ...VerificationKind) {
	fld := ff.fields[name]
	if fld == nil {
		panic(ErrFieldNotFound(name))
	}
	vf := fld.(interface{ setVerify(k ...VerificationKind) })
	vf.setVerify(vk...)
}

// # Implements:
//   - IFieldsBuilder
type fieldsBuilder struct {
	*fields
}

func makeFieldsBuilder(fields *fields) fieldsBuilder {
	return fieldsBuilder{
		fields: fields,
	}
}

func (fb *fieldsBuilder) AddDataField(name FieldName, data QName, required bool, constraints ...IConstraint) IFieldsBuilder {
	fb.fields.addDataField(name, data, required, constraints...)
	return fb
}

func (fb *fieldsBuilder) AddField(name FieldName, kind DataKind, required bool, constraints ...IConstraint) IFieldsBuilder {
	fb.fields.addField(name, kind, required, constraints...)
	return fb
}

func (fb *fieldsBuilder) AddRefField(name FieldName, required bool, ref ...QName) IFieldsBuilder {
	fb.fields.addRefField(name, required, ref...)
	return fb
}

func (fb *fieldsBuilder) SetFieldComment(name FieldName, comment ...string) IFieldsBuilder {
	fb.fields.setFieldComment(name, comment...)
	return fb
}

func (fb *fieldsBuilder) SetFieldVerify(name FieldName, vk ...VerificationKind) IFieldsBuilder {
	fb.fields.setFieldVerify(name, vk...)
	return fb
}

// # Implements:
//   - IRefField
type refField struct {
	field
	refs QNames
}

func newRefField(name FieldName, data IData, required bool, ref ...QName) *refField {
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
				refType := Record(t.App().Type, n)
				if refType == nil {
					err = errors.Join(err,
						ErrNotFound("%v reference field «%s» to unknown table «%v»", t, rf.Name(), n))
					continue
				}
			}
		}
	}
	return err
}

type nullFields struct{}

func (f *nullFields) Field(FieldName) IField       { return nil }
func (f *nullFields) FieldCount() int              { return 0 }
func (f *nullFields) Fields() []IField             { return []IField{} }
func (f *nullFields) RefField(FieldName) IRefField { return nil }
func (f *nullFields) RefFields() []IRefField       { return []IRefField{} }
func (f *nullFields) UserFieldCount() int          { return 0 }
func (f *nullFields) UserFields() []IField         { return []IField{} }

func (k VerificationKind) MarshalJSON() ([]byte, error) {
	var s string
	if k < VerificationKind_FakeLast {
		s = strconv.Quote(k.String())
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an VerificationKind in human-readable form, without "VerificationKind_" prefix,
// suitable for debugging or error messages
func (k VerificationKind) TrimString() string {
	const pref = "VerificationKind_"
	return strings.TrimPrefix(k.String(), pref)
}

func (k *VerificationKind) UnmarshalJSON(data []byte) (err error) {
	text := string(data)
	if t, err := strconv.Unquote(text); err == nil {
		text = t
		for v := VerificationKind(0); v < VerificationKind_FakeLast; v++ {
			if v.String() == text {
				*k = v
				return nil
			}
		}
	}

	uint8Val, err := utils.StringToUint8(text)
	if err == nil {
		*k = VerificationKind(uint8Val)
	}
	return err
}
