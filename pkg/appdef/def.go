/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// NullDef is used for return then definition	is not founded
var NullDef = newDef(nil, NullQName, DefKind_null)

// Implements IDef and IDefBuilder interfaces
type def struct {
	app               *appDef
	name              QName
	kind              DefKind
	fields            map[string]*field
	fieldsOrdered     []string
	containers        map[string]*container
	containersOrdered []string
	singleton         bool
}

func newDef(app *appDef, name QName, kind DefKind) *def {
	def := def{
		app:               app,
		name:              name,
		kind:              kind,
		fields:            make(map[string]*field),
		fieldsOrdered:     make([]string, 0),
		containers:        make(map[string]*container),
		containersOrdered: make([]string, 0),
	}
	def.makeSysFields()
	return &def
}

func (d *def) AddContainer(name string, contDef QName, minOccurs, maxOccurs Occurs) IDefBuilder {
	if name == NullName {
		panic(fmt.Errorf("empty container name: %w", ErrNameMissed))
	}
	if !IsSysContainer(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("invalid container name «%v»: %w", name, err))
		}
	}
	if d.Container(name) != nil {
		panic(fmt.Errorf("container «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}

	if maxOccurs == 0 {
		panic(fmt.Errorf("max occurs value (0) must be positive number: %w", ErrInvalidOccurs))
	}
	if maxOccurs < minOccurs {
		panic(fmt.Errorf("max occurs (%v) must be greater or equal to min occurs (%v): %w", maxOccurs, minOccurs, ErrInvalidOccurs))
	}

	if !d.Kind().ContainersAllowed() {
		panic(fmt.Errorf("definition «%s» kind «%v» does not allow containers: %w", d.QName(), d.Kind(), ErrInvalidDefKind))
	}
	if cd := d.app.DefByName(contDef); cd != nil {
		if !d.Kind().ContainerKindAvailable(cd.Kind()) {
			panic(fmt.Errorf("definition «%s» kind «%v» does not support child container kind «%v»: %w", d.QName(), d.Kind(), cd.Kind(), ErrInvalidDefKind))
		}
	}

	cont := newContainer(name, contDef, minOccurs, maxOccurs)
	d.containers[name] = &cont
	d.containersOrdered = append(d.containersOrdered, name)

	d.changed()

	return d
}

func (d *def) AddField(name string, kind DataKind, required bool) IDefBuilder {
	d.addField(name, kind, required, false)
	return d
}

func (d *def) App() IAppDef {
	return d.app
}

func (d *def) AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IDefBuilder {
	d.addField(name, kind, required, true, vk...)
	return d
}

func (d *def) Container(name string) Container {
	if c, ok := d.containers[name]; ok {
		return c
	}
	return nil
}

func (d *def) ContainerCount() int {
	return len(d.containersOrdered)
}

func (d *def) Containers(cb func(Container)) {
	for _, n := range d.containersOrdered {
		cb(d.Container(n))
	}
}

func (d *def) ContainerDef(contName string) IDef {
	if cont := d.Container(contName); cont != nil {
		return d.app.DefByName(cont.Def())
	}
	return nil
}

func (d *def) Field(name string) Field {
	if f, ok := d.fields[name]; ok {
		return f
	}
	return nil
}

func (d *def) FieldCount() int {
	return len(d.fieldsOrdered)
}

func (d *def) Fields(cb func(Field)) {
	for _, n := range d.fieldsOrdered {
		cb(d.Field(n))
	}
}

func (d *def) Kind() DefKind {
	return d.kind
}

func (d *def) QName() QName {
	return d.name
}

func (d *def) SetSingleton() {
	if d.Kind() != DefKind_CDoc {
		panic(fmt.Errorf("only CDocs can be singletons: %w", ErrInvalidDefKind))
	}
	d.singleton = true
	d.changed()
}

func (d *def) Singleton() bool {
	return d.singleton && (d.Kind() == DefKind_CDoc)
}

func (d *def) addField(name string, kind DataKind, required, verified bool, vk ...VerificationKind) {
	if name == NullName {
		panic(fmt.Errorf("empty field name: %w", ErrNameMissed))
	}
	if !IsSysField(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("field name «%v» is invalid: %w", name, err))
		}
	}
	if d.Field(name) != nil {
		if IsSysField(name) {
			return
		}
		panic(fmt.Errorf("field «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}
	if !d.Kind().FieldsAllowed() {
		panic(fmt.Errorf("definition «%s» kind «%v» does not allow fields: %w", d.QName(), d.Kind(), ErrInvalidDefKind))
	}
	if !d.Kind().DataKindAvailable(kind) {
		panic(fmt.Errorf("definition «%s» kind «%v» does not support fields kind «%v»: %w", d.QName(), d.Kind(), kind, ErrInvalidDataKind))
	}

	if verified && (len(vk) == 0) {
		panic(fmt.Errorf("missed verification kind for field «%v»: %w", name, ErrVerificationKindMissed))
	}

	fld := newField(name, kind, required, verified, vk...)
	d.fields[name] = fld
	d.fieldsOrdered = append(d.fieldsOrdered, name)

	d.changed()
}

func (d *def) changed() {
	if d.app != nil {
		d.app.changed()
	}
}

// clears fields and containers
func (d *def) clear() {
	d.fields = make(map[string]*field)
	d.fieldsOrdered = make([]string, 0)
	d.containers = make(map[string]*container)
	d.containersOrdered = make([]string, 0)
}

func (d *def) makeSysFields() {
	if d.Kind().HasSystemField(SystemField_QName) {
		d.AddField(SystemField_QName, DataKind_QName, true)
	}

	if d.Kind().HasSystemField(SystemField_ID) {
		d.AddField(SystemField_ID, DataKind_RecordID, true)
	}

	if d.Kind().HasSystemField(SystemField_ParentID) {
		d.AddField(SystemField_ParentID, DataKind_RecordID, true)
	}

	if d.Kind().HasSystemField(SystemField_Container) {
		d.AddField(SystemField_Container, DataKind_string, true)
	}

	if d.Kind().HasSystemField(SystemField_IsActive) {
		d.AddField(SystemField_IsActive, DataKind_bool, false)
	}
}
