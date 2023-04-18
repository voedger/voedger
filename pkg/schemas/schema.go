/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

// Schema describes the entity, such as document, record or view. Schema has fields and containers.
//
// Implements ISchema and ISchemaBuilder interfaces
type Schema struct {
	cache             *SchemasCache
	name              QName
	kind              SchemaKind
	props             SchemaKindProps
	fields            map[string]*Field
	fieldsOrdered     []string
	containers        map[string]*Container
	containersOrdered []string
	singleton         bool
}

func newSchema(cache *SchemasCache, name QName, kind SchemaKind) *Schema {
	schema := Schema{
		cache:             cache,
		name:              name,
		kind:              kind,
		props:             schemaKindProps[kind],
		fields:            make(map[string]*Field),
		fieldsOrdered:     make([]string, 0),
		containers:        make(map[string]*Container),
		containersOrdered: make([]string, 0),
	}
	schema.makeSysFields()
	return &schema
}

// Adds container specified name and occurs.
//
// # Panics:
//   - if name is empty,
//   - if container with name already exists,
//   - if invalid occurrences,
//   - if schema kind does not allow containers.
func (sch *Schema) AddContainer(name string, schema QName, minOccurs, maxOccurs Occurs) *Schema {
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

// Adds field specified name and kind.
//
// # Panics:
//   - if field is empty,
//   - if field with name is already exists,
//   - if schema kind not supports fields,
//   - if data kind is not allowed by schema kind.
func (sch *Schema) AddField(name string, kind DataKind, required bool) *Schema {
	sch.addField(name, kind, required, false)
	return sch
}

// Adds verified field specified name and kind.
//
// # Panics:
//   - if field is empty,
//   - if field with name is already exists,
//   - if schema kind not supports fields,
//   - if data kind is not allowed by schema kind.
func (sch *Schema) AddVerifiedField(name string, kind DataKind, required bool) *Schema {
	sch.addField(name, kind, required, true)
	return sch
}

// Finds container by name.
//
// Returns nil if not found.
func (sch *Schema) Container(name string) *Container {
	return sch.containers[name]
}

// Return container by index
func (sch *Schema) ContainerAt(idx int) *Container {
	return sch.Container(sch.containersOrdered[idx])
}

// Returns containers count
func (sch *Schema) ContainerCount() int {
	return len(sch.containersOrdered)
}

// Finds container schema by constinaer name.
//
// Returns nil if not found.
func (sch *Schema) ContainerSchema(contName string) *Schema {
	if cont := sch.Container(contName); cont != nil {
		return sch.cache.SchemaByName(cont.Schema())
	}
	return nil
}

// Enumerates all containers in add order.
func (sch *Schema) EnumContainers(cb func(*Container)) {
	for _, n := range sch.containersOrdered {
		cb(sch.Container(n))
	}
}

// Enumerates all fields in add order.
func (sch *Schema) EnumFields(cb func(*Field)) {
	for _, n := range sch.fieldsOrdered {
		cb(sch.Field(n))
	}
}

// Finds field by name.
//
// Returns nil if not found.
func (sch *Schema) Field(name string) *Field {
	return sch.fields[name]
}

// Returns fields count
func (sch *Schema) FieldCount() int {
	return len(sch.fieldsOrdered)
}

// Return field by index
func (sch *Schema) FieldAt(idx int) *Field {
	return sch.Field(sch.fieldsOrdered[idx])
}

// Schema kind properties.
func (sch *Schema) Props() SchemaKindProps {
	return sch.props
}

// Sets the singleton document flag for CDoc schemas.
//
// # Panics:
//   - if not CDoc schema.
func (sch *Schema) SetSingleton() {
	if sch.Kind() != istructs.SchemaKind_CDoc {
		panic(fmt.Errorf("only CDocs can be singletons: %w", ErrInvalidSchemaKind))
	}
	sch.singleton = true
}

// Returns is schema CDoc singleton
func (sch *Schema) Singleton() bool {
	return sch.singleton && (sch.Kind() == istructs.SchemaKind_CDoc)
}

func (sch *Schema) addField(name string, kind DataKind, required, verified bool) {
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
	sch.fields[name] = &fld
	sch.fieldsOrdered = append(sch.fieldsOrdered, name)
}

func (sch *Schema) clearFields() {
	sch.fields = make(map[string]*Field)
	sch.fieldsOrdered = make([]string, 0)
}

func (sch *Schema) makeSysFields() {
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

// —————————— istructs.ISchema ——————————

func (sch *Schema) Kind() SchemaKind {
	return sch.kind
}

func (sch *Schema) QName() QName {
	return sch.name
}

func (sch *Schema) Fields(cb func(fieldName string, kind DataKind)) {
	sch.EnumFields(func(f *Field) { cb(f.Name(), f.DataKind()) })
}

func (sch *Schema) ForEachField(cb func(istructs.IFieldDescr)) {
	sch.EnumFields(func(f *Field) { cb(f) })
}

func (sch *Schema) Containers(cb func(containerName string, schema QName)) {
	sch.EnumContainers(func(c *Container) { cb(c.Name(), c.Schema()) })
}

func (sch *Schema) ForEachContainer(cb func(istructs.IContainerDescr)) {
	sch.EnumContainers(func(c *Container) { cb(c) })
}
