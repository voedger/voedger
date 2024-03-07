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

	v.fields = makeFields(app, TypeKind_ViewRecord)
	v.fields.makeSysFields()
	v.key = newViewKey(v)
	v.value = newViewValue(v)

	app.appendType(v)

	return v
}

func (v *view) Key() IViewKey {
	return v.key
}

func (v *view) Value() IViewValue {
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
//   - IViewBuilder
type viewBuilder struct {
	*view
	typeBuilder
	key *viewKeyBuilder
	val *viewValueBuilder
}

func newViewBuilder(v *view) *viewBuilder {
	return &viewBuilder{
		view:        v,
		typeBuilder: makeTypeBuilder(&v.typ),
		key:         newViewKeyBuilder(v.key),
		val:         newViewValueBuilder(v.value),
	}
}

func (vb *viewBuilder) Key() IViewKeyBuilder { return vb.key }

func (vb *viewBuilder) Value() IViewValueBuilder { return vb.val }

// # Implements:
//   - IViewKey
type viewKey struct {
	view   *view
	fields // all key fields, include partition key and clustering columns
	pkey   *viewPartKey
	ccols  *viewClustCols
}

func newViewKey(view *view) *viewKey {
	key := &viewKey{
		view:   view,
		fields: makeFields(view.typ.app, TypeKind_ViewRecord),
		pkey:   newViewPartKey(view),
		ccols:  newViewClustCols(view),
	}

	return key
}

func (key *viewKey) PartKey() IViewPartKey {
	return key.pkey
}

func (key *viewKey) ClustCols() IViewClustCols {
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
//   - IViewKeyBuilder
type viewKeyBuilder struct {
	*viewKey
	pkey  *viewPartKeyBuilder
	ccols *viewClustColsBuilder
}

func newViewKeyBuilder(key *viewKey) *viewKeyBuilder {
	return &viewKeyBuilder{
		viewKey: key,
		pkey:    newViewPartKeyBuilder(key.pkey),
		ccols:   newViewClustColsBuilder(key.ccols),
	}
}

func (kb *viewKeyBuilder) ClustCols() IViewClustColsBuilder { return kb.ccols }

func (kb *viewKeyBuilder) PartKey() IViewPartKeyBuilder { return kb.pkey }

// # Implements:
//   - IViewPartKey
type viewPartKey struct {
	view *view
	fields
}

func newViewPartKey(v *view) *viewPartKey {
	pKey := &viewPartKey{
		view:   v,
		fields: makeFields(v.typ.app, TypeKind_ViewRecord),
	}
	return pKey
}

func (pk *viewPartKey) addDataField(name string, dataType QName, constraints ...IConstraint) {
	d := pk.view.typ.app.Data(dataType)
	if d == nil {
		panic(fmt.Errorf("%v: view partition key field «%s» has unknown data type «%s»: %w", pk.view.QName(), name, dataType, ErrNameNotFound))
	}
	if k := d.DataKind(); !k.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", pk.view.QName(), name, k.TrimString(), ErrInvalidDataKind))
	}

	pk.view.addDataField(name, dataType, true, constraints...)
	pk.view.key.addDataField(name, dataType, true, constraints...)
	pk.fields.addDataField(name, dataType, true, constraints...)
}

func (pk *viewPartKey) addField(name string, kind DataKind, constraints ...IConstraint) {
	if !kind.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", pk.view.QName(), name, kind.TrimString(), ErrInvalidDataKind))
	}

	pk.view.addField(name, kind, true, constraints...)
	pk.view.key.addField(name, kind, true, constraints...)
	pk.fields.addField(name, kind, true, constraints...)
}

func (pk *viewPartKey) addRefField(name string, ref ...QName) {
	pk.view.addRefField(name, true, ref...)
	pk.view.key.addRefField(name, true, ref...)
	pk.fields.addRefField(name, true, ref...)
}

func (pk *viewPartKey) isPartKey() {}

func (pk *viewPartKey) setFieldComment(name string, comment ...string) {
	pk.view.setFieldComment(name, comment...)
	pk.view.key.setFieldComment(name, comment...)
	pk.fields.setFieldComment(name, comment...)
}

// Validates view partition key
func (pk *viewPartKey) validate() error {
	if pk.FieldCount() == 0 {
		return fmt.Errorf("%v: view partition key can not to be empty: %w", pk.view.QName(), ErrFieldsMissed)
	}
	return nil
}

// # Implements:
//   - IViewPartKeyBuilder
type viewPartKeyBuilder struct {
	*viewPartKey
}

func newViewPartKeyBuilder(viewPartKey *viewPartKey) *viewPartKeyBuilder {
	return &viewPartKeyBuilder{
		viewPartKey: viewPartKey,
	}
}

func (pkb *viewPartKeyBuilder) AddDataField(name string, dataType QName, constraints ...IConstraint) IViewPartKeyBuilder {
	pkb.viewPartKey.addDataField(name, dataType, constraints...)
	return pkb
}

func (pkb *viewPartKeyBuilder) AddField(name string, kind DataKind, constraints ...IConstraint) IViewPartKeyBuilder {
	pkb.viewPartKey.addField(name, kind, constraints...)
	return pkb
}

func (pkb *viewPartKeyBuilder) AddRefField(name string, ref ...QName) IViewPartKeyBuilder {
	pkb.viewPartKey.addRefField(name, ref...)
	return pkb
}

func (pkb *viewPartKeyBuilder) SetFieldComment(name string, comment ...string) IViewPartKeyBuilder {
	pkb.viewPartKey.setFieldComment(name, comment...)
	return pkb
}

// # Implements:
//   - IViewClustCols
type viewClustCols struct {
	view *view
	fields
	varField string
}

func newViewClustCols(v *view) *viewClustCols {
	cc := &viewClustCols{
		view:   v,
		fields: makeFields(v.typ.app, TypeKind_ViewRecord),
	}
	return cc
}

func (cc *viewClustCols) addDataField(name string, dataType QName, constraints ...IConstraint) {
	d := cc.app.Data(dataType)
	if d == nil {
		panic(fmt.Errorf("%v: data type «%v» not found: %w", cc.view, dataType, ErrNameNotFound))
	}

	cc.panicIfVarFieldDuplication(name, d.DataKind())

	cc.view.addDataField(name, dataType, false, constraints...)
	cc.view.key.addDataField(name, dataType, false, constraints...)
	cc.fields.addDataField(name, dataType, false, constraints...)
}

func (cc *viewClustCols) addField(name string, kind DataKind, constraints ...IConstraint) {
	cc.panicIfVarFieldDuplication(name, kind)

	cc.view.addField(name, kind, false, constraints...)
	cc.view.key.addField(name, kind, false, constraints...)
	cc.fields.addField(name, kind, false, constraints...)
}

func (cc *viewClustCols) addRefField(name string, ref ...QName) {
	cc.panicIfVarFieldDuplication(name, DataKind_RecordID)

	cc.view.addRefField(name, false, ref...)
	cc.view.key.addRefField(name, false, ref...)
	cc.fields.addRefField(name, false, ref...)
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

func (cc *viewClustCols) setFieldComment(name string, comment ...string) {
	cc.view.setFieldComment(name, comment...)
	cc.view.key.setFieldComment(name, comment...)
	cc.fields.setFieldComment(name, comment...)
}

// Validates view clustering columns
func (cc *viewClustCols) validate() (err error) {
	if cc.FieldCount() == 0 {
		return fmt.Errorf("%v: view clustering columns can not to be empty: %w", cc.view.QName(), ErrFieldsMissed)
	}

	return nil
}

// # Implements:
//   - IViewClustColsBuilder
type viewClustColsBuilder struct {
	*viewClustCols
}

func newViewClustColsBuilder(viewClustCols *viewClustCols) *viewClustColsBuilder {
	return &viewClustColsBuilder{
		viewClustCols: viewClustCols,
	}
}

func (ccb *viewClustColsBuilder) AddDataField(name string, dataType QName, constraints ...IConstraint) IViewClustColsBuilder {
	ccb.viewClustCols.addDataField(name, dataType, constraints...)
	return ccb
}

func (ccb *viewClustColsBuilder) AddField(name string, kind DataKind, constraints ...IConstraint) IViewClustColsBuilder {
	ccb.viewClustCols.addField(name, kind, constraints...)
	return ccb
}

func (ccb *viewClustColsBuilder) AddRefField(name string, ref ...QName) IViewClustColsBuilder {
	ccb.viewClustCols.addRefField(name, ref...)
	return ccb
}

func (ccb *viewClustColsBuilder) SetFieldComment(name string, comment ...string) IViewClustColsBuilder {
	ccb.viewClustCols.setFieldComment(name, comment...)
	return ccb
}

// # Implements:
//   - IViewValue
type viewValue struct {
	view *view
	fields
}

func newViewValue(v *view) *viewValue {
	val := &viewValue{
		view:   v,
		fields: makeFields(v.typ.app, TypeKind_ViewRecord),
	}
	val.fields.makeSysFields()
	return val
}

func (v *viewValue) addDataField(name string, dataType QName, required bool, constraints ...IConstraint) {
	v.view.addDataField(name, dataType, required, constraints...)
	v.fields.addDataField(name, dataType, required, constraints...)
}

func (v *viewValue) AddField(name string, kind DataKind, required bool, constraints ...IConstraint) {
	v.view.addField(name, kind, required, constraints...)
	v.fields.addField(name, kind, required, constraints...)
}

func (v *viewValue) addRefField(name string, required bool, ref ...QName) {
	v.view.addRefField(name, required, ref...)
	v.fields.addRefField(name, required, ref...)
}

func (v *viewValue) isViewValue() {}

func (v *viewValue) setFieldComment(name string, comment ...string) {
	v.view.setFieldComment(name, comment...)
	v.fields.setFieldComment(name, comment...)
}

func (v *viewValue) setFieldVerify(name string, vk ...VerificationKind) {
	v.view.setFieldVerify(name, vk...)
	v.fields.setFieldVerify(name, vk...)
}

// Validates view value
func (v *viewValue) validate() error {
	return nil
}

// # Implements:
//   - IViewValueBuilder
type viewValueBuilder struct {
	*viewValue
}

func newViewValueBuilder(viewValue *viewValue) *viewValueBuilder {
	return &viewValueBuilder{
		viewValue: viewValue,
	}
}

func (vb *viewValueBuilder) AddDataField(name string, dataType QName, required bool, constraints ...IConstraint) IFieldsBuilder {
	vb.viewValue.addDataField(name, dataType, required, constraints...)
	return vb
}

func (vb *viewValueBuilder) AddField(name string, kind DataKind, required bool, constraints ...IConstraint) IFieldsBuilder {
	vb.viewValue.AddField(name, kind, required, constraints...)
	return vb
}

func (vb *viewValueBuilder) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	vb.viewValue.addRefField(name, required, ref...)
	return vb
}

func (vb *viewValueBuilder) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	vb.viewValue.setFieldComment(name, comment...)
	return vb
}

func (vb *viewValueBuilder) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
	vb.viewValue.setFieldVerify(name, vk...)
	return vb
}
