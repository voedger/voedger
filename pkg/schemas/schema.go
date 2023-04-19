/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

// Implements ISchema and ISchemaBuilder interfaces
type schema struct {
	cache             *schemasCache
	name              QName
	kind              SchemaKind
	props             SchemaKindProps
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
		props:             schemaKindProps[kind],
		fields:            make(map[string]*field),
		fieldsOrdered:     make([]string, 0),
		containers:        make(map[string]*container),
		containersOrdered: make([]string, 0),
	}
	schema.makeSysFields()
	return &schema
}

func (sch *schema) AddContainer(name string, schema QName, minOccurs, maxOccurs Occurs) SchemaBuilder {
	if name == "" {
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

	if !sch.Props().ContainersAllowed() {
		panic(fmt.Errorf("schema «%s» kind «%v» does not allow containers: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}
	if contSchema := sch.cache.SchemaByName(schema); contSchema != nil {
		if !sch.Props().ContainerKindAvailable(contSchema.Kind()) {
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

func (sch *schema) Props() SchemaKindProps {
	return sch.props
}

func (sch *schema) QName() QName {
	return sch.name
}

func (sch *schema) SetSingleton() {
	if sch.Kind() != istructs.SchemaKind_CDoc {
		panic(fmt.Errorf("only CDocs can be singletons: %w", ErrInvalidSchemaKind))
	}
	sch.singleton = true
}

func (sch *schema) Singleton() bool {
	return sch.singleton && (sch.Kind() == istructs.SchemaKind_CDoc)
}

func (sch *schema) addField(name string, kind DataKind, required, verified bool) {
	if name == "" {
		panic(fmt.Errorf("empty field name: %w", ErrNameMissed))
	}
	if sch.Field(name) != nil {
		panic(fmt.Errorf("field «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}
	// TODO: check name is valid
	if !sch.Props().FieldsAllowed() {
		panic(fmt.Errorf("schema «%s» kind «%v» does not allow fields: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}
	if !sch.Props().DataKindAvailable(kind) {
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
	if sch.Props().HasSystemField(istructs.SystemField_QName) {
		sch.AddField(istructs.SystemField_QName, istructs.DataKind_QName, true)
	}

	if sch.Props().HasSystemField(istructs.SystemField_ID) {
		sch.AddField(istructs.SystemField_ID, istructs.DataKind_RecordID, true)
	}

	if sch.Props().HasSystemField(istructs.SystemField_ParentID) {
		sch.AddField(istructs.SystemField_ParentID, istructs.DataKind_RecordID, true)
	}

	if sch.Props().HasSystemField(istructs.SystemField_Container) {
		sch.AddField(istructs.SystemField_Container, istructs.DataKind_string, true)
	}

	if sch.Props().HasSystemField(istructs.SystemField_IsActive) {
		sch.AddField(istructs.SystemField_IsActive, istructs.DataKind_bool, false)
	}
}

// ————— istrucst.ISchema —————
func (sch *schema) Fields(cb func(fieldName string, kind istructs.DataKindType)) {
	sch.EnumFields(func(f Field) { cb(f.Name(), f.DataKind()) })
}

func (sch *schema) ForEachField(cb func(field istructs.IFieldDescr)) {
	sch.EnumFields(func(f Field) { cb(f) })
}

func (sch *schema) Containers(cb func(containerName string, schema QName)) {
	sch.EnumContainers(func(c Container) { cb(c.Name(), c.Schema()) })
}

func (sch *schema) ForEachContainer(cb func(cont istructs.IContainerDescr)) {
	sch.EnumContainers(func(c Container) { cb(c) })
}
