/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// View
//
// # Implements:
//   - IView
//   - IViewBuilder
type view struct {
	typ
	fields // all fields, include key and value
	key    *viewKey
	value  *viewValue
}

func newView(app *appDef, name QName) *view {
	v := &view{
		typ: makeType(app, name, TypeKind_ViewRecord),
	}

	v.fields = makeFields(v)
	v.fields.makeSysFields(TypeKind_ViewRecord)
	v.key = newViewKey(v)
	v.value = newViewValue(v)

	app.appendType(v)

	return v
}

func (v *view) Key() IViewKey {
	return v.key
}

func (v *view) KeyBuilder() IViewKeyBuilder {
	return v.key
}

func (v *view) Value() IViewValue {
	return v.value
}

func (v *view) ValueBuilder() IViewValueBuilder {
	return v.value
}

// Validates view
func (v *view) Validate() error {
	return errors.Join(
		v.key.validate(),
		v.value.validate(),
	)
}

// View key.
//
// # Implements:
//   - IViewKey
//   - IViewKeyBuilder
type viewKey struct {
	view   *view
	fields // all key fields, include partition key and clustering columns
	pkey   *viewPartKey
	ccols  *viewClustCols
}

func newViewKey(v *view) *viewKey {
	key := &viewKey{
		view:   v,
		fields: makeFields(v),
		pkey:   newViewPartKey(v),
		ccols:  newViewClustCols(v),
	}

	return key
}

func (key *viewKey) PartKey() IViewPartKey {
	return key.pkey
}

func (key *viewKey) PartKeyBuilder() IViewPartKeyBuilder {
	return key.pkey
}

func (key *viewKey) ClustCols() IViewClustCols {
	return key.ccols
}

func (key *viewKey) ClustColsBuilder() IViewClustColsBuilder {
	return key.ccols
}

// Validates value key
func (key *viewKey) validate() error {
	return errors.Join(
		key.pkey.validate(),
		key.ccols.validate(),
	)
}

// View partition key.
//
// # Implements:
//   - IViewPartKey
//   - IViewPartKeyBuilder
type viewPartKey struct {
	view *view
	fields
}

func newViewPartKey(v *view) *viewPartKey {
	pKey := &viewPartKey{
		view:   v,
		fields: makeFields(v),
	}
	return pKey
}

func (pk *viewPartKey) AddField(name string, kind DataKind, comment ...string) IViewPartKeyBuilder {
	if !kind.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", pk.view.QName(), name, kind.TrimString(), ErrInvalidDataKind))
	}

	pk.view.AddField(name, kind, true, comment...)
	pk.view.key.AddField(name, kind, true, comment...)
	pk.fields.AddField(name, kind, true, comment...)
	return pk
}

func (pk *viewPartKey) AddRefField(name string, ref ...QName) IViewPartKeyBuilder {
	pk.view.AddRefField(name, true, ref...)
	pk.view.key.AddRefField(name, true, ref...)
	pk.fields.AddRefField(name, true, ref...)
	return pk
}

func (pk *viewPartKey) SetFieldComment(name string, comment ...string) IViewPartKeyBuilder {
	pk.view.SetFieldComment(name, comment...)
	pk.view.key.SetFieldComment(name, comment...)
	pk.fields.SetFieldComment(name, comment...)
	return pk
}

// Validates view partition key
func (pk *viewPartKey) validate() error {
	if pk.FieldCount() == 0 {
		return fmt.Errorf("%v: view partition key can not to be empty: %w", pk.view.QName(), ErrFieldsMissed)
	}
	return nil
}

// View clustering columns
//
// # Implements:
//   - IViewClustCols
//   - IViewClustColsBuilder
type viewClustCols struct {
	view *view
	fields
	varField string
}

func newViewClustCols(v *view) *viewClustCols {
	cc := &viewClustCols{
		view:   v,
		fields: makeFields(v),
	}
	return cc
}

func (cc *viewClustCols) AddField(name string, kind DataKind, comment ...string) IViewClustColsBuilder {
	cc.panicIfVarFieldDuplication(name, kind)

	cc.view.AddField(name, kind, false, comment...)
	cc.view.key.AddField(name, kind, false, comment...)
	cc.fields.AddField(name, kind, false, comment...)
	return cc
}

func (cc *viewClustCols) AddRefField(name string, ref ...QName) IViewClustColsBuilder {
	cc.panicIfVarFieldDuplication(name, DataKind_RecordID)

	cc.view.AddRefField(name, false, ref...)
	cc.view.key.AddRefField(name, false, ref...)
	cc.fields.AddRefField(name, false, ref...)
	return cc
}

func (cc *viewClustCols) AddStringField(name string, maxLen uint16) IViewClustColsBuilder {
	cc.panicIfVarFieldDuplication(name, DataKind_string)

	cc.view.AddStringField(name, false, MaxLen(maxLen))
	cc.view.key.AddStringField(name, false, MaxLen(maxLen))
	cc.fields.AddStringField(name, false, MaxLen(maxLen))
	return cc
}

func (cc *viewClustCols) AddBytesField(name string, maxLen uint16) IViewClustColsBuilder {
	cc.panicIfVarFieldDuplication(name, DataKind_bytes)

	cc.view.AddBytesField(name, false, MaxLen(maxLen))
	cc.view.key.AddBytesField(name, false, MaxLen(maxLen))
	cc.fields.AddBytesField(name, false, MaxLen(maxLen))
	return cc
}

func (cc *viewClustCols) SetFieldComment(name string, comment ...string) IViewClustColsBuilder {
	cc.view.SetFieldComment(name, comment...)
	cc.view.key.SetFieldComment(name, comment...)
	cc.fields.SetFieldComment(name, comment...)
	return cc
}

// Panics if variable length field already exists
func (cc *viewClustCols) panicIfVarFieldDuplication(name string, kind DataKind) {
	if len(cc.varField) > 0 {
		panic(fmt.Errorf("%v: view clustering column already has a field of variable length «%s», you can not add a field «%s» after it: %w", cc.view.QName(), cc.varField, name, ErrInvalidDataKind))
	}
	if !kind.IsFixed() {
		cc.varField = name
	}
}

// Validates view clustering columns
func (cc *viewClustCols) validate() (err error) {
	if cc.FieldCount() == 0 {
		return fmt.Errorf("%v: view clustering columns can not to be empty: %w", cc.view.QName(), ErrFieldsMissed)
	}

	return nil
}

// View value
//
// # Implements:
//   - IViewValue
//   - IViewValueBuilder
type viewValue struct {
	view *view
	fields
}

func newViewValue(v *view) *viewValue {
	val := &viewValue{
		view:   v,
		fields: makeFields(v),
	}
	val.fields.makeSysFields(TypeKind_ViewRecord)
	return val
}

func (v *viewValue) AddField(name string, kind DataKind, required bool, comment ...string) IFieldsBuilder {
	v.view.AddField(name, kind, required, comment...)
	v.fields.AddField(name, kind, required, comment...)
	return v
}

func (v *viewValue) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	v.view.AddRefField(name, required, ref...)
	v.fields.AddRefField(name, required, ref...)
	return v
}

func (v *viewValue) AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.AddStringField(name, required, restricts...)
	v.fields.AddStringField(name, required, restricts...)
	return v
}

func (v *viewValue) AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.AddBytesField(name, required, restricts...)
	v.fields.AddBytesField(name, required, restricts...)
	return v
}

func (v *viewValue) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	v.view.AddVerifiedField(name, kind, required, vk...)
	v.fields.AddVerifiedField(name, kind, required, vk...)
	return v
}

func (v *viewValue) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	v.view.SetFieldComment(name, comment...)
	v.fields.SetFieldComment(name, comment...)
	return v
}

func (v *viewValue) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
	v.view.SetFieldVerify(name, vk...)
	v.fields.SetFieldVerify(name, vk...)
	return v
}

// Validates view value
func (v *viewValue) validate() error {
	return nil
}
