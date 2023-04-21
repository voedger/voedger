/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"
)

// NullSchema is used for return then schema	is not founded
var NullSchema = newSchema(nil, NullQName, SchemaKind_null)

// Implements ISchema and ISchemaBuilder interfaces
type schema struct {
	cache             *schemasCache
	name              QName
	kind              SchemaKind
	fields            map[string]*field
	fieldsOrdered     []string
	containers        map[string]*container
	containersOrdered []string
	singleton         bool
}

func newSchema(cache *schemasCache, name QName, kind SchemaKind) *schema {
	schema := schema{
		cache:             cache,
		name:              name,
		kind:              kind,
		fields:            make(map[string]*field),
		fieldsOrdered:     make([]string, 0),
		containers:        make(map[string]*container),
		containersOrdered: make([]string, 0),
	}
	schema.makeSysFields()
	return &schema
}

func (sch *schema) AddContainer(name string, schema QName, minOccurs, maxOccurs Occurs) SchemaBuilder {
	if name == NullName {
		panic(fmt.Errorf("empty container name: %w", ErrNameMissed))
	}
	if !IsSysContainer(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("invalid container name «%v»: %w", name, err))
		}
	}
	if sch.Container(name) != nil {
		panic(fmt.Errorf("container «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}

	if maxOccurs == 0 {
		panic(fmt.Errorf("max occurs value (0) must be positive number: %w", ErrInvalidOccurs))
	}
	if maxOccurs < minOccurs {
		panic(fmt.Errorf("max occurs (%v) must be greater or equal to min occurs (%v): %w", maxOccurs, minOccurs, ErrInvalidOccurs))
	}

	if !sch.Kind().ContainersAllowed() {
		panic(fmt.Errorf("schema «%s» kind «%v» does not allow containers: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}
	if contSchema := sch.cache.SchemaByName(schema); contSchema != nil {
		if !sch.Kind().ContainerKindAvailable(contSchema.Kind()) {
			panic(fmt.Errorf("schema «%s» kind «%v» does not support child container kind «%v»: %w", sch.QName(), sch.Kind(), contSchema.Kind(), ErrInvalidSchemaKind))
		}
	}

	cont := newContainer(name, schema, minOccurs, maxOccurs)
	sch.containers[name] = &cont
	sch.containersOrdered = append(sch.containersOrdered, name)

	return sch
}

func (sch *schema) AddField(name string, kind DataKind, required bool) SchemaBuilder {
	sch.addField(name, kind, required, false)
	return sch
}

func (sch *schema) AddVerifiedField(name string, kind DataKind, required bool) SchemaBuilder {
	sch.addField(name, kind, required, true)
	return sch
}

func (sch *schema) Cache() SchemaCache {
	return sch.cache
}

func (sch *schema) Container(name string) Container {
	if c, ok := sch.containers[name]; ok {
		return c
	}
	return nil
}

func (sch *schema) ContainerCount() int {
	return len(sch.containersOrdered)
}

func (sch *schema) EnumContainers(cb func(Container)) {
	for _, n := range sch.containersOrdered {
		cb(sch.Container(n))
	}
}
func (sch *schema) ContainerSchema(contName string) Schema {
	if cont := sch.Container(contName); cont != nil {
		return sch.cache.SchemaByName(cont.Schema())
	}
	return nil
}

func (sch *schema) Field(name string) Field {
	if f, ok := sch.fields[name]; ok {
		return f
	}
	return nil
}

func (sch *schema) EnumFields(cb func(Field)) {
	for _, n := range sch.fieldsOrdered {
		cb(sch.Field(n))
	}
}

func (sch *schema) FieldCount() int {
	return len(sch.fieldsOrdered)
}

func (sch *schema) Kind() SchemaKind {
	return sch.kind
}

func (sch *schema) QName() QName {
	return sch.name
}

func (sch *schema) SetSingleton() {
	if sch.Kind() != SchemaKind_CDoc {
		panic(fmt.Errorf("only CDocs can be singletons: %w", ErrInvalidSchemaKind))
	}
	sch.singleton = true
}

func (sch *schema) Singleton() bool {
	return sch.singleton && (sch.Kind() == SchemaKind_CDoc)
}

func (sch *schema) addField(name string, kind DataKind, required, verified bool) {
	if name == NullName {
		panic(fmt.Errorf("empty field name: %w", ErrNameMissed))
	}
	if sch.Field(name) != nil {
		panic(fmt.Errorf("field «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}
	// TODO: check name is valid
	if !sch.Kind().FieldsAllowed() {
		panic(fmt.Errorf("schema «%s» kind «%v» does not allow fields: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}
	if !sch.Kind().DataKindAvailable(kind) {
		panic(fmt.Errorf("schema «%s» kind «%v» does not support fields kind «%v»: %w", sch.QName(), sch.Kind(), kind, ErrInvalidDataKind))
	}

	fld := newField(name, kind, required, verified)
	sch.fields[name] = fld
	sch.fieldsOrdered = append(sch.fieldsOrdered, name)
}

// clears fields and containers
func (sch *schema) clear() {
	sch.fields = make(map[string]*field)
	sch.fieldsOrdered = make([]string, 0)
	sch.containers = make(map[string]*container)
	sch.containersOrdered = make([]string, 0)
}

func (sch *schema) makeSysFields() {
	if sch.Kind().HasSystemField(SystemField_QName) {
		sch.AddField(SystemField_QName, DataKind_QName, true)
	}

	if sch.Kind().HasSystemField(SystemField_ID) {
		sch.AddField(SystemField_ID, DataKind_RecordID, true)
	}

	if sch.Kind().HasSystemField(SystemField_ParentID) {
		sch.AddField(SystemField_ParentID, DataKind_RecordID, true)
	}

	if sch.Kind().HasSystemField(SystemField_Container) {
		sch.AddField(SystemField_Container, DataKind_string, true)
	}

	if sch.Kind().HasSystemField(SystemField_IsActive) {
		sch.AddField(SystemField_IsActive, DataKind_bool, false)
	}
}
