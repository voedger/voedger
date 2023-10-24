/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

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

	v.fields = makeFields(app, v)
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
		fields: makeFields(v.typ.app, v),
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
		fields: makeFields(v.typ.app, v),
	}
	return pKey
}

func (pk *viewPartKey) AddDataField(name string, dataType QName, constraints ...IConstraint) IViewPartKeyBuilder {
	d := pk.view.typ.app.Data(dataType)
	if d == nil {
		panic(fmt.Errorf("%v: view partition key field «%s» has unknown data type «%s»: %w", pk.view.QName(), name, dataType, ErrNameNotFound))
	}
	if k := d.DataKind(); !k.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", pk.view.QName(), name, k.TrimString(), ErrInvalidDataKind))
	}

	pk.view.AddDataField(name, dataType, true, constraints...)
	pk.view.key.AddDataField(name, dataType, true, constraints...)
	pk.fields.AddDataField(name, dataType, true, constraints...)
	return pk
}

func (pk *viewPartKey) AddField(name string, kind DataKind, constraints ...IConstraint) IViewPartKeyBuilder {
	if !kind.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", pk.view.QName(), name, kind.TrimString(), ErrInvalidDataKind))
	}

	pk.view.AddField(name, kind, true, constraints...)
	pk.view.key.AddField(name, kind, true, constraints...)
	pk.fields.AddField(name, kind, true, constraints...)
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

func (pk *viewPartKey) isPartKey() {}

// Validates view partition key
func (pk *viewPartKey) validate() error {
	if pk.FieldCount() == 0 {
		return fmt.Errorf("%v: view partition key can not to be empty: %w", pk.view.QName(), ErrFieldsMissed)
	}
	return nil
}

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
		fields: makeFields(v.typ.app, v),
	}
	return cc
}

func (cc *viewClustCols) AddBytesField(name string, maxLen uint16) IViewClustColsBuilder {
	return cc.AddField(name, DataKind_bytes, MaxLen(maxLen))
}

func (cc *viewClustCols) AddDataField(name string, dataType QName, constraints ...IConstraint) IViewClustColsBuilder {
	d := cc.app.Data(dataType)
	if d == nil {
		panic(fmt.Errorf("%v: data type «%v» not found: %w", cc.view, dataType, ErrNameNotFound))
	}

	cc.panicIfVarFieldDuplication(name, d.DataKind())

	cc.view.AddDataField(name, dataType, false, constraints...)
	cc.view.key.AddDataField(name, dataType, false, constraints...)
	cc.fields.AddDataField(name, dataType, false, constraints...)
	return cc
}

func (cc *viewClustCols) AddField(name string, kind DataKind, constraints ...IConstraint) IViewClustColsBuilder {
	cc.panicIfVarFieldDuplication(name, kind)

	cc.view.AddField(name, kind, false, constraints...)
	cc.view.key.AddField(name, kind, false, constraints...)
	cc.fields.AddField(name, kind, false, constraints...)
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
	return cc.AddField(name, DataKind_string, MaxLen(maxLen))
}

func (cc *viewClustCols) SetFieldComment(name string, comment ...string) IViewClustColsBuilder {
	cc.view.SetFieldComment(name, comment...)
	cc.view.key.SetFieldComment(name, comment...)
	cc.fields.SetFieldComment(name, comment...)
	return cc
}

func (cc *viewClustCols) isClustCols() {}

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
		fields: makeFields(v.typ.app, v),
	}
	val.fields.makeSysFields(TypeKind_ViewRecord)
	return val
}

func (v *viewValue) AddBytesField(name string, required bool, constraints ...IConstraint) IFieldsBuilder {
	return v.AddField(name, DataKind_bytes, required, constraints...)
}

func (v *viewValue) AddDataField(name string, dataType QName, required bool, constraints ...IConstraint) IFieldsBuilder {
	v.view.AddDataField(name, dataType, required, constraints...)
	v.fields.AddDataField(name, dataType, required, constraints...)
	return v
}

func (v *viewValue) AddField(name string, kind DataKind, required bool, constraints ...IConstraint) IFieldsBuilder {
	v.view.AddField(name, kind, required, constraints...)
	v.fields.AddField(name, kind, required, constraints...)
	return v
}

func (v *viewValue) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	v.view.AddRefField(name, required, ref...)
	v.fields.AddRefField(name, required, ref...)
	return v
}

func (v *viewValue) AddStringField(name string, required bool, constraints ...IConstraint) IFieldsBuilder {
	return v.AddField(name, DataKind_string, required, constraints...)
}

func (v *viewValue) isViewValue() {}

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
