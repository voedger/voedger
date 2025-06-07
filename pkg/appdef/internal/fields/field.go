/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
 */

package fields

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
)

// # Supports:
//   - appdef.IField
type Field struct {
	comments.WithComments
	name        appdef.FieldName
	data        appdef.IData
	required    bool
	verifiable  bool
	verify      map[appdef.VerificationKind]bool
	constraints map[appdef.ConstraintKind]appdef.IConstraint
}

func MakeField(name appdef.FieldName, data appdef.IData, required bool, c ...string) Field {
	f := Field{
		WithComments: comments.MakeWithComments(c...),
		name:         name,
		data:         data,
		required:     required,
		verifiable:   false,
		constraints:  data.Constraints(true),
	}
	return f
}

func NewField(name appdef.FieldName, data appdef.IData, required bool, comments ...string) *Field {
	f := MakeField(name, data, required, comments...)
	return &f
}

func (fld *Field) Constraints() map[appdef.ConstraintKind]appdef.IConstraint {
	return fld.constraints
}

func (fld *Field) Data() appdef.IData { return fld.data }

func (fld *Field) DataKind() appdef.DataKind { return fld.Data().DataKind() }

func (fld *Field) IsFixedWidth() bool { return fld.DataKind().IsFixed() }

func (fld *Field) IsSys() bool { return appdef.IsSysField(fld.Name()) }

func (fld *Field) Name() appdef.FieldName { return fld.name }

func (fld *Field) Required() bool { return fld.required }

func (fld Field) String() string {
	return fmt.Sprintf("%s-field «%s»", fld.DataKind().TrimString(), fld.Name())
}

func (fld *Field) Verifiable() bool { return fld.verifiable }

func (fld *Field) VerificationKind(vk appdef.VerificationKind) bool {
	return fld.verifiable && fld.verify[vk]
}

func (fld *Field) setVerify(k ...appdef.VerificationKind) {
	fld.verify = make(map[appdef.VerificationKind]bool)
	for _, kind := range k {
		fld.verify[kind] = true
	}
	fld.verifiable = len(fld.verify) > 0
}

// # Supports:
//   - appdef.IFields
type WithFields struct {
	ws            appdef.IWorkspace
	typeKind      appdef.TypeKind
	fields        map[appdef.FieldName]interface{}
	fieldsOrdered []appdef.IField
	refFields     []appdef.IRefField
	userFields    []appdef.IField
}

// Makes new fields instance
func MakeWithFields(ws appdef.IWorkspace, typeKind appdef.TypeKind) WithFields {
	ff := WithFields{
		ws:            ws,
		typeKind:      typeKind,
		fields:        make(map[appdef.FieldName]interface{}),
		fieldsOrdered: make([]appdef.IField, 0),
		refFields:     make([]appdef.IRefField, 0),
		userFields:    make([]appdef.IField, 0)}
	return ff
}

func (ff WithFields) Field(name appdef.FieldName) appdef.IField {
	if ff, ok := ff.fields[name]; ok {
		return ff.(appdef.IField)
	}
	return nil
}

func (ff WithFields) FieldCount() int { return len(ff.fieldsOrdered) }

func (ff WithFields) Fields() []appdef.IField { return ff.fieldsOrdered }

func (ff WithFields) RefField(name appdef.FieldName) (rf appdef.IRefField) {
	if fld := ff.Field(name); fld != nil {
		if fld.DataKind() == appdef.DataKind_RecordID {
			if fld, ok := fld.(appdef.IRefField); ok {
				rf = fld
			}
		}
	}
	return rf
}

// Makes system fields. Called after making structures fields
func (ff *WithFields) MakeSysFields() {
	if exists, required := ff.typeKind.HasSystemField(appdef.SystemField_QName); exists {
		ff.addField(appdef.SystemField_QName, appdef.DataKind_QName, required)
	}

	if exists, required := ff.typeKind.HasSystemField(appdef.SystemField_ID); exists {
		ff.addField(appdef.SystemField_ID, appdef.DataKind_RecordID, required)
	}

	if exists, required := ff.typeKind.HasSystemField(appdef.SystemField_ParentID); exists {
		ff.addField(appdef.SystemField_ParentID, appdef.DataKind_RecordID, required)
	}

	if exists, required := ff.typeKind.HasSystemField(appdef.SystemField_Container); exists {
		ff.addField(appdef.SystemField_Container, appdef.DataKind_string, required)
	}

	if exists, required := ff.typeKind.HasSystemField(appdef.SystemField_IsActive); exists {
		ff.addField(appdef.SystemField_IsActive, appdef.DataKind_bool, required)
	}
}

func (ff WithFields) RefFields() []appdef.IRefField { return ff.refFields }

func (ff WithFields) UserFields() []appdef.IField { return ff.userFields }

func (ff WithFields) UserFieldCount() int { return len(ff.userFields) }

func (ff *WithFields) addDataField(name appdef.FieldName, data appdef.QName, required bool, constraints ...appdef.IConstraint) {
	d := appdef.Data(ff.ws.Type, data)
	if d == nil {
		panic(appdef.ErrTypeNotFound(data))
	}
	if len(constraints) > 0 {
		d = datas.NewAnonymousData(ff.ws, d.DataKind(), data, constraints...)
	}
	f := NewField(name, d, required)
	ff.appendField(name, f)
}

func (ff *WithFields) addField(name appdef.FieldName, kind appdef.DataKind, required bool, constraints ...appdef.IConstraint) {
	d := appdef.SysData(ff.ws.Type, kind)
	if d == nil {
		panic(appdef.ErrNotFound("system data type for data kind «%s»", kind.TrimString()))
	}
	if len(constraints) > 0 {
		d = datas.NewAnonymousData(ff.ws, d.DataKind(), d.QName(), constraints...)
	}
	f := NewField(name, d, required)
	ff.appendField(name, f)
}

func (ff *WithFields) addRefField(name appdef.FieldName, required bool, ref ...appdef.QName) {
	d := appdef.SysData(ff.ws.Type, appdef.DataKind_RecordID)
	f := NewRefField(name, d, required, ref...)
	ff.appendField(name, f)
}

// Appends specified field.
//
// # Panics:
//   - if field name is empty,
//   - if field with specified name is already exists
//   - if user field name is invalid
//   - if user field data kind is not allowed by structured type kind
func (ff *WithFields) appendField(name appdef.FieldName, fld interface{}) {
	if name == appdef.NullName {
		panic(appdef.ErrMissed("field name"))
	}
	if ff.Field(name) != nil {
		panic(appdef.ErrAlreadyExists("field «%v»", name))
	}
	if len(ff.fields) >= appdef.MaxTypeFieldCount {
		panic(appdef.ErrTooMany("fields, maximum is %d", appdef.MaxTypeFieldCount))
	}

	f := fld.(appdef.IField)

	if !appdef.IsSysField(name) {
		if ok, err := appdef.ValidFieldName(name); !ok {
			panic(fmt.Errorf("field name «%v» is invalid: %w", name, err))
		}
		dk := f.DataKind()
		if (ff.typeKind != appdef.TypeKind_null) && !ff.typeKind.FieldKindAvailable(dk) {
			panic(appdef.ErrIncompatible("data kind «%s» with fields of «%v»", dk.TrimString(), ff.typeKind.TrimString()))
		}
	}

	ff.fields[name] = fld
	ff.fieldsOrdered = append(ff.fieldsOrdered, f)
	if !appdef.IsSysField(name) {
		ff.userFields = append(ff.userFields, f)
	}

	if rf, ok := fld.(appdef.IRefField); ok {
		ff.refFields = append(ff.refFields, rf)
	}
}

func (ff *WithFields) setFieldComment(name appdef.FieldName, comment ...string) {
	fld := ff.fields[name]
	if fld == nil {
		panic(appdef.ErrFieldNotFound(name))
	}
	switch f := fld.(type) {
	case *Field:
		comments.SetComment(&f.WithComments, comment...)
	case *RefField:
		comments.SetComment(&f.WithComments, comment...)
	}
}

func (ff *WithFields) setFieldVerify(name appdef.FieldName, vk ...appdef.VerificationKind) {
	fld := ff.fields[name]
	if fld == nil {
		panic(appdef.ErrFieldNotFound(name))
	}
	vf := fld.(interface {
		setVerify(k ...appdef.VerificationKind)
	})
	vf.setVerify(vk...)
}

// # Supports:
//   - appdef.IFieldsBuilder
type FieldsBuilder struct {
	*WithFields
}

func MakeFieldsBuilder(fields *WithFields) FieldsBuilder {
	return FieldsBuilder{WithFields: fields}
}

func (fb *FieldsBuilder) AddDataField(name appdef.FieldName, data appdef.QName, required bool, constraints ...appdef.IConstraint) appdef.IFieldsBuilder {
	fb.WithFields.addDataField(name, data, required, constraints...)
	return fb
}

func (fb *FieldsBuilder) AddField(name appdef.FieldName, kind appdef.DataKind, required bool, constraints ...appdef.IConstraint) appdef.IFieldsBuilder {
	fb.WithFields.addField(name, kind, required, constraints...)
	return fb
}

func (fb *FieldsBuilder) AddRefField(name appdef.FieldName, required bool, ref ...appdef.QName) appdef.IFieldsBuilder {
	fb.WithFields.addRefField(name, required, ref...)
	return fb
}

func (fb *FieldsBuilder) SetFieldComment(name appdef.FieldName, comment ...string) appdef.IFieldsBuilder {
	fb.WithFields.setFieldComment(name, comment...)
	return fb
}

func (fb *FieldsBuilder) SetFieldVerify(name appdef.FieldName, vk ...appdef.VerificationKind) appdef.IFieldsBuilder {
	fb.WithFields.setFieldVerify(name, vk...)
	return fb
}

// # Supports:
//   - appdef.IRefField
type RefField struct {
	Field
	refs appdef.QNames
}

func NewRefField(name appdef.FieldName, data appdef.IData, required bool, ref ...appdef.QName) *RefField {
	f := &RefField{
		Field: MakeField(name, data, required),
		refs:  appdef.QNames{},
	}
	f.refs.Add(ref...)
	return f
}

func (f RefField) Ref(n appdef.QName) bool {
	l := len(f.refs)
	if l == 0 {
		return true // any ref available
	}
	return f.refs.Contains(n)
}

func (f RefField) Refs() []appdef.QName { return f.refs }

// Validates specified fields.
//
// # Validation:
//   - every RefField must refer to known types,
//   - every referenced by RefField type must be record type
func ValidateTypeFields(t appdef.IType) (err error) {
	if ff, ok := t.(appdef.IWithFields); ok {
		// resolve reference types
		for _, rf := range ff.RefFields() {
			for _, n := range rf.Refs() {
				refType := appdef.Record(t.App().Type, n)
				if refType == nil {
					err = errors.Join(err,
						appdef.ErrNotFound("%v reference field «%s» to unknown table «%v»", t, rf.Name(), n))
					continue
				}
			}
		}
	}
	return err
}

func AddDataField(fields *WithFields, name appdef.FieldName, data appdef.QName, required bool, constraints ...appdef.IConstraint) {
	fields.addDataField(name, data, required, constraints...)
}

func AddField(fields *WithFields, name appdef.FieldName, kind appdef.DataKind, required bool, constraints ...appdef.IConstraint) {
	fields.addField(name, kind, required, constraints...)
}

func AddRefField(fields *WithFields, name appdef.FieldName, required bool, ref ...appdef.QName) {
	fields.addRefField(name, required, ref...)
}

func SetFieldComment(fields *WithFields, name appdef.FieldName, comment ...string) {
	fields.setFieldComment(name, comment...)
}

func SetFieldVerify(fields *WithFields, name appdef.FieldName, vk ...appdef.VerificationKind) {
	fields.setFieldVerify(name, vk...)
}
