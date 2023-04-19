/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// TODO: this types must moved here from istructs package
type (
	QName      = istructs.QName
	SchemaKind = istructs.SchemaKindType
	DataKind   = istructs.DataKindType
	Occurs     = istructs.ContainerOccursType
)

// Application schemas
type SchemaCache interface {
	istructs.ISchemas

	// Enumerates all schemas from cache.
	EnumSchemas(func(Schema))

	// Return count of schemas.
	SchemaCount() int

	// Returns schema by name.
	//
	// Returns nil if not found.
	SchemaByName(name QName) Schema
}

// Application schemas builder
type SchemaCacheBuilder interface {
	SchemaCache

	// Adds new schema specified name and kind.
	//
	// # Panics:
	//   - if name is empty (istructs.NullQName),
	//   - if schema with name already exists.
	Add(name QName, kind SchemaKind) SchemaBuilder

	// Adds new schemas for view.
	AddView(QName) ViewBuilder

	// Must be called after all schemas added. Validates schemas and returns builded schemas or error
	Build() (SchemaCache, error)
}

// Schema describes the entity, such as document, record or view. Schema has fields and containers.
type Schema interface {
	istructs.ISchema

	// Parent cache
	Cache() SchemaCache

	// Schema qualified name.
	QName() QName

	// Schema kind.
	Kind() SchemaKind

	// Schema kind properties.
	Props() SchemaKindProps

	// Finds field by name.
	//
	// Returns nil if not found.
	Field(name string) Field

	// Enumerates all fields in add order.
	EnumFields(func(Field))

	// Returns fields count
	FieldCount() int

	// Finds container by name.
	//
	// Returns nil if not found.
	Container(name string) Container

	// Enumerates all containers in add order.
	EnumContainers(func(Container))

	// Returns containers count
	ContainerCount() int

	// Finds container schema by constinaer name.
	//
	// Returns nil if not found.
	ContainerSchema(name string) Schema

	// Returns is schema CDoc singleton
	Singleton() bool

	validate() error
}

// Schema builder
type SchemaBuilder interface {
	Schema

	// Adds field specified name and kind.
	//
	// # Panics:
	//   - if field is empty,
	//   - if field with name is already exists,
	//   - if schema kind not supports fields,
	//   - if data kind is not allowed by schema kind.
	AddField(name string, kind DataKind, required bool) SchemaBuilder

	// Adds verified field specified name and kind.
	//
	// # Panics:
	//   - if field is empty,
	//   - if field with name is already exists,
	//   - if schema kind not supports fields,
	//   - if data kind is not allowed by schema kind.
	AddVerifiedField(name string, kind DataKind, required bool) SchemaBuilder

	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if container with name already exists,
	//   - if invalid occurrences,
	//   - if schema kind does not allow containers.
	AddContainer(name string, schema QName, min, max Occurs) SchemaBuilder

	// Sets the singleton document flag for CDoc schemas.
	//
	// # Panics:
	//   - if not CDoc schema.
	SetSingleton()

	clear()
}

// View builder
type ViewBuilder interface {
	// Returns view name
	Name() QName

	// Schema returns view schema
	Schema() SchemaBuilder

	// PartKeySchema: returns view partition key schema
	PartKeySchema() SchemaBuilder

	// ClustColsSchema returns view clustering columns schema
	ClustColsSchema() SchemaBuilder

	// FullKeySchema returns view full key (partition key + clustering columns) schema
	FullKeySchema() SchemaBuilder

	// ValueSchema returns view value schema
	ValueSchema() SchemaBuilder

	// AddPartField adds specisified field to view partition key schema. Fields is always required
	AddPartField(name string, kind DataKind) ViewBuilder

	// AddClustColumn adds specisified field to view clustering columns schema. Fields is optional
	AddClustColumn(name string, kind DataKind) ViewBuilder

	// AddValueField adds specisified field to view value schema
	AddValueField(name string, kind DataKind, required bool) ViewBuilder
}

// Describe single field.
type Field interface {
	// Returns field name
	Name() string

	// Returns data kind for field
	DataKind() DataKind

	// Returns is field required
	Required() bool

	// Returns is field verifable
	Verifiable() bool

	// Returns is field has fixed width data kind
	IsFixedWidth() bool

	// Returns is field system
	IsSys() bool
}

// Describes single inclusion of child schema in parent schema.
type Container interface {
	// Returns name of container
	Name() string

	// Returns schema name of container
	Schema() QName

	// Returns minimum occurs
	MinOccurs() Occurs

	// Returns maximum occurs
	MaxOccurs() Occurs

	// Returns is container system
	IsSys() bool
}
