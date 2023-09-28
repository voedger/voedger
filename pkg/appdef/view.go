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
	containers
	key   *viewKey
	value *viewValue
}

func newView(app *appDef, name QName) *view {
	v := &view{typ: makeType(app, name, TypeKind_ViewRecord)}
	v.containers = makeContainers(v)

	v.key = newViewKey(app, name)
	v.value = newViewValue(app, name)
	v.
		AddContainer(SystemContainer_ViewKey, v.key.QName(), 1, 1).
		AddContainer(SystemContainer_ViewValue, v.value.QName(), 1, 1)

	app.appendType(v)

	return v
}

func (v *view) Key() IViewKey {
	return v.key
}

func (v *view) Value() IViewValue {
	return v.value
}

func (v *view) panicIfFieldDuplication(name string) {
	check := func(f IFields) {
		if fld := f.Field(name); fld != nil {
			panic(fmt.Errorf("field «%s» already exists in view «%v»: %w", name, v.QName(), ErrNameUniqueViolation))
		}
	}

	check(v.key)
	check(v.value)
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
	typ
	fields
	containers
	pkey  *viewPartKey
	ccols *viewClustCols
}

func newViewKey(app *appDef, viewName QName) *viewKey {
	key := &viewKey{typ: makeType(app, ViewKeyDefName(viewName), TypeKind_ViewRecord_Key)}
	key.fields = makeFields(key)
	key.containers = makeContainers(key)

	key.pkey = newViewPartKey(app, ViewPartitionKeyDefName(viewName))
	key.ccols = newViewClustCols(app, ViewClusteringColumnsDefName(viewName))

	key.
		AddContainer(SystemContainer_ViewPartitionKey, key.pkey.QName(), 1, 1).
		AddContainer(SystemContainer_ViewClusteringCols, key.ccols.QName(), 1, 1)

	app.appendType(key)
	return key
}

func (key *viewKey) Partition() IViewPartKey {
	return key.pkey
}

func (key *viewKey) ClustCols() IViewClustCols {
	return key.ccols
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
	typ
	fields
}

func newViewPartKey(app *appDef, name QName) *viewPartKey {
	pKey := &viewPartKey{typ: makeType(app, name, TypeKind_ViewRecord_PartitionKey)}
	pKey.fields = makeFields(pKey)
	app.appendType(pKey)
	return pKey
}

// Validates view partition key
func (pk *viewPartKey) Validate() error {
	if pk.FieldCount() == 0 {
		return fmt.Errorf("%v: view partition key can not to be empty: %w", pk.QName(), ErrFieldsMissed)
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
	k.view.panicIfFieldDuplication(name)
	k.view.key.pkey.AddField(name, kind, true, comment...)
	k.view.key.AddField(name, kind, true, comment...)
	return k
}

func (k *viewPartKeyBuilder) AddRefField(name string, ref ...QName) IViewPartKeyBuilder {
	k.view.panicIfFieldDuplication(name)
	k.view.key.pkey.AddRefField(name, true, ref...)
	k.view.key.AddRefField(name, true, ref...)
	return k
}

func (k *viewPartKeyBuilder) SetFieldComment(name string, comment ...string) IViewPartKeyBuilder {
	k.view.key.pkey.SetFieldComment(name, comment...)
	k.view.key.SetFieldComment(name, comment...)
	return k
}

// View clustering columns
//
// # Implements:
//   - IViewClustCols
type viewClustCols struct {
	typ
	fields
}

func newViewClustCols(app *appDef, name QName) *viewClustCols {
	cc := &viewClustCols{typ: makeType(app, name, TypeKind_ViewRecord_ClusteringColumns)}
	cc.fields = makeFields(cc)
	app.appendType(cc)
	return cc
}

// Validates view clustering columns
func (cc *viewClustCols) Validate() (err error) {
	if cc.FieldCount() == 0 {
		return fmt.Errorf("%v: view clustering columns can not to be empty: %w", cc.QName(), ErrFieldsMissed)
	}

	idx, cnt := 0, cc.FieldCount()
	cc.Fields(func(fld IField) {
		idx++
		if idx == cnt {
			return // last field may be any kind
		}
		if !fld.IsFixedWidth() {
			err = errors.Join(err,
				fmt.Errorf("%v: only last view clustering column field can be variable length; not last field «%s» has variable length type «%s»: %w", cc.QName(), fld.Name(), fld.DataKind().TrimString(), ErrInvalidDataKind))
		}
	})

	return err
}

// View clustering columns builder
//
// # Implements:
//   - IViewClustColsBuilder
type viewClustColsBuilder struct {
	view *view
}

func newViewClustColsBuilder(v *view) *viewClustColsBuilder {
	return &viewClustColsBuilder{v}
}

func (c *viewClustColsBuilder) AddField(name string, kind DataKind, comment ...string) IViewClustColsBuilder {
	c.view.panicIfFieldDuplication(name)
	c.view.key.ccols.AddField(name, kind, false, comment...)
	c.view.key.AddField(name, kind, false, comment...)
	return c
}

func (c *viewClustColsBuilder) AddRefField(name string, ref ...QName) IViewClustColsBuilder {
	c.view.panicIfFieldDuplication(name)
	c.view.key.ccols.AddRefField(name, false, ref...)
	c.view.key.AddRefField(name, false, ref...)
	return c
}

func (c *viewClustColsBuilder) AddStringField(name string, maxLen uint16) IViewClustColsBuilder {
	c.view.panicIfFieldDuplication(name)
	c.view.key.ccols.AddStringField(name, false, MaxLen(maxLen))
	c.view.key.AddStringField(name, false, MaxLen(maxLen))
	return c
}

func (c *viewClustColsBuilder) AddBytesField(name string, maxLen uint16) IViewClustColsBuilder {
	c.view.panicIfFieldDuplication(name)
	c.view.key.ccols.AddBytesField(name, false, MaxLen(maxLen))
	c.view.key.AddBytesField(name, false, MaxLen(maxLen))
	return c
}

func (c *viewClustColsBuilder) SetFieldComment(name string, comment ...string) IViewClustColsBuilder {
	c.view.key.ccols.SetFieldComment(name, comment...)
	c.view.key.SetFieldComment(name, comment...)
	return c
}

// View value
//
// # Implements:
//   - IViewValue
type viewValue struct {
	typ
	fields
}

func newViewValue(app *appDef, viewName QName) *viewValue {
	val := &viewValue{typ: makeType(app, ViewValueDefName(viewName), TypeKind_ViewRecord_Value)}
	val.fields = makeFields(val)
	app.appendType(val)
	return val
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
	v.view.panicIfFieldDuplication(name)
	v.view.value.AddField(name, kind, required, comment...)
	return v
}

func (v *viewValueBuilder) AddRefField(name string, required bool, ref ...QName) IFieldsBuilder {
	v.view.panicIfFieldDuplication(name)
	v.view.value.AddRefField(name, required, ref...)
	return v
}

func (v *viewValueBuilder) AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.panicIfFieldDuplication(name)
	v.view.value.AddStringField(name, required, restricts...)
	return v
}

func (v *viewValueBuilder) AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder {
	v.view.panicIfFieldDuplication(name)
	v.view.value.AddBytesField(name, required, restricts...)
	return v
}

func (v *viewValueBuilder) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder {
	v.view.panicIfFieldDuplication(name)
	v.view.value.AddVerifiedField(name, kind, required, vk...)
	return v
}

func (v *viewValueBuilder) SetFieldComment(name string, comment ...string) IFieldsBuilder {
	v.view.value.SetFieldComment(name, comment...)
	return v
}

func (v *viewValueBuilder) SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder {
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
