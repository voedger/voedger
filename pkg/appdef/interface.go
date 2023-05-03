/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Qualified name
//
// <pkg>.<entity>
//
// Ref to qname.go for constants and methods
type QName struct {
	pkg    string
	entity string
}

// Definition kind enumeration.
//
// Ref. def-kind.go for constants and methods
type DefKind uint8

// Data kind enumeration
//
// Ref. data-kind.go for constants and methods
type DataKind uint8

// Field Verification kind
//
// Ref. verification-king.go for constants and methods
type VerificationKind uint8

// Numeric with OccursUnbounded value
//
// Ref. occurs.go for constants and methods
type Occurs uint16

// Application definition.
//
// Ref to apdef.go for implementation
type IAppDef interface {
	// Returns definition by name.
	//
	// If not found empty definition with DefKind_null is returned
	Def(name QName) IDef

	// Returns definition by name.
	//
	// Returns nil if not found.
	DefByName(name QName) IDef

	// Return count of definitions.
	DefCount() int

	// Enumerates all application definitions.
	Defs(func(IDef))
}

// Application definition builder
//
// Ref to appdef.go for implementation
type IAppDefBuilder interface {
	IAppDef

	// Adds new definition specified name and kind.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists,
	//   - if kind is not structure.
	// # Structures are:
	//	- DefKind_GDoc, DefKind_CDoc, DefKind_ODoc, DefKind_WDoc,
	//	-	DefKind_GRecord, DefKind_CRecord, DefKind_ORecord, DefKind_WRecord
	//	- DefKind_Object and DefKind_Element.
	AddStruct(name QName, kind DefKind) IDefBuilder

	// Adds new definitions for view.
	AddView(QName) IViewBuilder

	// Must be called after all definitions added. Validates and returns builded application definition or error
	Build() (IAppDef, error)

	// Has changes since last success build
	HasChanges() bool
}

// Definition describes the entity, such as document, record or view. Definitions may have fields and containers.
//
// Ref to def.go for implementation
type IDef interface {
	// Parent cache
	App() IAppDef

	// Definition qualified name.
	QName() QName

	// Definition kind.
	Kind() DefKind

	// Finds field by name.
	//
	// Returns nil if not found.
	Field(name string) IField

	// Returns fields count
	FieldCount() int

	// Enumerates all fields in add order.
	Fields(func(IField))

	// Finds container by name.
	//
	// Returns nil if not found.
	Container(name string) IContainer

	// Returns containers count
	ContainerCount() int

	// Enumerates all containers in add order.
	Containers(func(IContainer))

	// Finds container definition by constainer name.
	//
	// If not found empty definition with DefKind_null is returned
	ContainerDef(name string) IDef

	// Returns is definition CDoc singleton
	Singleton() bool

	// Return unique by name.
	//
	// Returns nil if not unique found
	Unique(name string) []IField

	// Return uniques count
	UniqueCount() int

	// Enumerates all uniques
	Uniques(func(name string, fields []IField))
}

// Definition builder
//
// Ref to def.go for implementation
type IDefBuilder interface {
	IDef

	// Adds field specified name and kind.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if definition kind not supports fields,
	//   - if data kind is not allowed by definition kind.
	AddField(name string, kind DataKind, required bool) IDefBuilder

	// Adds verified field specified name and kind.
	//
	// # Panics:
	//   - if field name is empty,
	//   - if field name is invalid,
	//   - if field with name is already exists,
	//   - if definition kind not supports fields,
	//   - if data kind is not allowed by definition kind,
	//   - if no verification kinds are specified
	AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IDefBuilder

	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if container with name already exists,
	//   - if invalid occurrences,
	//   - if definition kind does not allow containers,
	//   - if container definition kind is not compatable with definition kind.
	AddContainer(name string, def QName, min, max Occurs) IDefBuilder

	// Adds new unique with specified name and fields set.
	// If name is omitted, then default name is used, e.g. `unique01`.
	//
	// # Panics:
	//   - if unique name is invalid,
	//   - if unique with name is already exists,
	//   - if definition kind is not supports uniques,
	//   - if fields set is empty,
	//   - if fields has duplicates,
	//   - if fields set is already exists or overlaps with an existing more general unique,
	//   - if some field not found.
	AddUnique(name string, fields []string) IDefBuilder

	// Sets the singleton document flag for CDoc.
	//
	// # Panics:
	//   - if not CDoc definition.
	SetSingleton()
}

// View builder
//
// Ref to view.go for implementation
type IViewBuilder interface {
	// Returns view name
	Name() QName

	// Returns view definition
	Def() IDefBuilder

	// Returns view partition key definition
	PartKeyDef() IDefBuilder

	// Returns view clustering columns definition
	ClustColsDef() IDefBuilder

	// Returns view value definition
	ValueDef() IDefBuilder

	// AddPartField adds specisified field to view partition key definition. Fields is always required
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddPartField(name string, kind DataKind) IViewBuilder

	// AddClustColumn adds specisified field to view clustering columns definition. Fields is optional
	//
	// # Panics:
	//	- if field already exists in partition key or value fields.
	AddClustColumn(name string, kind DataKind) IViewBuilder

	// AddValueField adds specisified field to view value definition
	//
	// # Panics:
	//	- if field already exists in partition key or clustering columns fields.
	AddValueField(name string, kind DataKind, required bool) IViewBuilder
}

// Describe single field.
//
// Ref to field.go for constants and implementation
type IField interface {
	// Returns field name
	Name() string

	// Returns data kind for field
	DataKind() DataKind

	// Returns is field required
	Required() bool

	// Returns is field verifable
	Verifiable() bool

	// Returns is field verifable by specified verification kind
	VerificationKind(VerificationKind) bool

	// Returns is field has fixed width data kind
	IsFixedWidth() bool

	// Returns is field system
	IsSys() bool
}

// Describes single inclusion of child definition in parent definition.
//
// Ref to container.go for constants and implementation
type IContainer interface {
	// Returns name of container
	Name() string

	// Returns definition name of container
	Def() QName

	// Returns minimum occurs
	MinOccurs() Occurs

	// Returns maximum occurs
	MaxOccurs() Occurs

	// Returns is container system
	IsSys() bool
}
