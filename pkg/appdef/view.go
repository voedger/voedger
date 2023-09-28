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
type view struct {
	typ
	comment
	fields // all fields, include key and value
	key    *viewKey
	value  *viewValue
}

func newView(app *appDef, name QName) *view {
	v := &view{
		typ: makeType(app, name, TypeKind_ViewRecord),
	}

	v.fields = makeFields(v)
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

// View builder
//
// # Implements:
//   - IViewBuilder
type viewBuilder struct {
	view  *view
	key   *viewKeyBuilder
	value *viewValueBuilder
}

func newViewBuilder(app *appDef, name QName) *viewBuilder {
	v := newView(app, name)
	return &viewBuilder{
		view:  v,
		key:   newViewKeyBuilder(v),
		value: newViewValueBuilder(v),
	}
}

func (v *viewBuilder) Key() IViewKeyBuilder {
	return v.key
}

func (v *viewBuilder) SetComment(line ...string) {
	v.view.SetComment(line...)
}

func (v *viewBuilder) Value() IViewValueBuilder {
	return v.value
}

// View key.
//
// # Implements:
//   - IViewKey
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

func (key *viewKey) Partition() IViewPartKey {
	return key.pkey
}

func (key *viewKey) ClustCols() IViewClustCols {
	return key.ccols
}

func (key *viewKey) validate() error {
	return errors.Join(
		key.pkey.validate(),
		key.ccols.validate(),
	)
}

// View key builder
//
// # Implements:
//   - IViewKeyBuilder
type viewKeyBuilder struct {
	view  *view
	pkey  *viewPartKeyBuilder
	ccols *viewClustColsBuilder
}

func newViewKeyBuilder(v *view) *viewKeyBuilder {
	k := &viewKeyBuilder{
		view:  v,
		pkey:  newViewPartKeyBuilder(v),
		ccols: newViewClustColsBuilder(v),
	}
	return k
}

func (k *viewKeyBuilder) Partition() IViewPartKeyBuilder {
	return k.pkey
}

func (k *viewKeyBuilder) ClustCols() IViewClustColsBuilder {
	return k.ccols
}

// View partition key.
//
// # Implements:
//   - IViewPartKey
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

// Validates view partition key
func (pk *viewPartKey) validate() error {
	if pk.FieldCount() == 0 {
		return fmt.Errorf("%v: view partition key can not to be empty: %w", pk.view.QName(), ErrFieldsMissed)
	}
	return nil
}

// View partition key builder.
//
// # Implements:
//   - IViewPartKeyBuilder
type viewPartKeyBuilder struct {
	view *view
}

func newViewPartKeyBuilder(v *view) *viewPartKeyBuilder {
	return &viewPartKeyBuilder{v}
}

func (k *viewPartKeyBuilder) AddField(name string, kind DataKind, comment ...string) IViewPartKeyBuilder {
	if !kind.IsFixed() {
		panic(fmt.Errorf("%v: view partition key field «%s» has variable length type «%s»: %w", k.view.QName(), name, kind.TrimString(), ErrInvalidDataKind))
	}

	k.view.AddField(name, kind, true, comment...)
	k.view.key.pkey.AddField(name, kind, true, comment...)
	k.view.key.AddField(name, kind, true, comment...)
	return k
}

func (k *viewPartKeyBuilder) AddRefField(name string, ref ...QName) IViewPartKeyBuilder {
	k.view.AddRefField(name, true, ref...)
	k.view.key.pkey.AddRefField(name, true, ref...)
	k.view.key.AddRefField(name, true, ref...)
	return k
}

func (k *viewPartKeyBuilder) SetFieldComment(name string, comment ...string) IViewPartKeyBuilder {
	k.view.SetFieldComment(name, comment...)
	k.view.key.pkey.SetFieldComment(name, comment...)
	k.view.key.SetFieldComment(name, comment...)
	return k
}

// View clustering columns
//
// # Implements:
//   - IViewClustCols
type viewClustCols struct {
	view *view
	fields
}

func newViewClustCols(v *view) *viewClustCols {
	cc := &viewClustCols{
		view:   v,
		fields: makeFields(v),
	}
	return cc
}

// Validates view clustering columns
func (cc *viewClustCols) validate() (err error) {
	if cc.FieldCount() == 0 {
		return fmt.Errorf("%v: view clustering columns can not to be empty: %w", cc.view.QName(), ErrFieldsMissed)
	}

	return nil
}

// View clustering columns builder
//
// # Implements:
//   - IViewClustColsBuilder
type viewClustColsBuilder struct {
	view     *view
	varField string
}

func newViewClustColsBuilder(v *view) *viewClustColsBuilder {
	return &viewClustColsBuilder{
		view:     v,
		varField: "",
	}
}

func (c *viewClustColsBuilder) AddField(name string, kind DataKind, comment ...string) IViewClustColsBuilder {
	c.panicIfVarFieldDuplication(name, kind)

	c.view.AddField(name, kind, false, comment...)
	c.view.key.ccols.AddField(name, kind, false, comment...)
	c.view.key.AddField(name, kind, false, comment...)
	return c
}

func (c *viewClustColsBuilder) AddRefField(name string, ref ...QName) IViewClustColsBuilder {
	c.panicIfVarFieldDuplication(name, DataKind_RecordID)

	c.view.AddRefField(name, false, ref...)
	c.view.key.ccols.AddRefField(name, false, ref...)
	c.view.key.AddRefField(name, false, ref...)
	return c
}

func (c *viewClustColsBuilder) AddStringField(name string, maxLen uint16) IViewClustColsBuilder {
	c.panicIfVarFieldDuplication(name, DataKind_string)

	c.view.AddStringField(name, false, MaxLen(maxLen))
	c.view.key.ccols.AddStringField(name, false, MaxLen(maxLen))
	c.view.key.AddStringField(name, false, MaxLen(maxLen))
	return c
}

func (c *viewClustColsBuilder) AddBytesField(name string, maxLen uint16) IViewClustColsBuilder {
	c.panicIfVarFieldDuplication(name, DataKind_bytes)

	c.view.AddBytesField(name, false, MaxLen(maxLen))
	c.view.key.ccols.AddBytesField(name, false, MaxLen(maxLen))
	c.view.key.AddBytesField(name, false, MaxLen(maxLen))
	return c
}

func (c *viewClustColsBuilder) SetFieldComment(name string, comment ...string) IViewClustColsBuilder {
	c.view.SetFieldComment(name, comment...)
	c.view.key.ccols.SetFieldComment(name, comment...)
	c.view.key.SetFieldComment(name, comment...)
	return c
}

func (c *viewClustColsBuilder) panicIfVarFieldDuplication(name string, kind DataKind) {
	if len(c.varField) > 0 {
		panic(fmt.Errorf("%v: view clustering column already has a field of variable length «%s», you can not add a field «%s» after it: %w", c.view.QName(), c.varField, name, ErrInvalidDataKind))
	}
	if !kind.IsFixed() {
		c.varField = name
	}
}

// View value
//
// # Implements:
//   - IViewValue
type viewValue struct {
	view *view
	fields
}

func newViewValue(v *view) *viewValue {
	val := &viewValue{
		view:   v,
		fields: makeFields(v),
	}
	return val
}

// Validates view value
func (v *viewValue) validate() error {
	return nil
}

// View value builder
//
// # Implements:
//   - IViewValueBuilder
type viewValueBuilder struct {
	view *view
}

func newViewValueBuilder(v *view) *viewValueBuilder {
	return &viewValueBuilder{v}
}

func (v *viewValueBuilder) AddField(name string, kind DataKind, required bool, comment ...string) IFieldsBuilder {
	v.view.AddField(name, kind, required, comment...)
	v.view.value.AddField(name, kind, required, comment...)
	return v
}

func (v *viewValueBuilder) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	v.view.AddRefField(name, required, ref...)
	v.view.value.AddRefField(name, required, ref...)
	return v
}

func (v *viewValueBuilder) AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.AddStringField(name, required, restricts...)
	v.view.value.AddStringField(name, required, restricts...)
	return v
}

func (v *viewValueBuilder) AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.AddBytesField(name, required, restricts...)
	v.view.value.AddBytesField(name, required, restricts...)
	return v
}

func (v *viewValueBuilder) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	v.view.AddVerifiedField(name, kind, required, vk...)
	v.view.value.AddVerifiedField(name, kind, required, vk...)
	return v
}

func (v *viewValueBuilder) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	v.view.SetFieldComment(name, comment...)
	v.view.value.SetFieldComment(name, comment...)
	return v
}

func (v *viewValueBuilder) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
	v.view.SetFieldVerify(name, vk...)
	v.view.value.SetFieldVerify(name, vk...)
	return v
}

// Returns partition key type name for specified view
func ViewPartitionKeyDefName(viewName QName) QName {
	const suffix = "_PartitionKey"
	return suffixedQName(viewName, suffix)
}

// Returns clustering columns type name for specified view
func ViewClusteringColumnsDefName(viewName QName) QName {
	const suffix = "_ClusteringColumns"
	return suffixedQName(viewName, suffix)
}

// Returns full key type name for specified view
func ViewKeyDefName(viewName QName) QName {
	const suffix = "_FullKey"
	return suffixedQName(viewName, suffix)
}

// Returns value type name for specified view
func ViewValueDefName(viewName QName) QName {
	const suffix = "_Value"
	return suffixedQName(viewName, suffix)
}

// Appends suffix to QName entity name and returns new QName
func suffixedQName(name QName, suffix string) QName {
	return NewQName(name.Pkg(), name.Entity()+suffix)
}
