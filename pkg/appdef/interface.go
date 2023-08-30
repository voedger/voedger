/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "regexp"

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

// Data kind enumeration.
//
// Ref. data-kind.go for constants and methods
type DataKind uint8

// Field Verification kind.
//
// Ref. verification-king.go for constants and methods
type VerificationKind uint8

// Numeric with OccursUnbounded value.
//
// Ref. occurs.go for constants and methods
type Occurs uint16

// Extension engine kind enumeration.
//
// Ref. to extension-engine-kind.go for constants and methods
type ExtensionEngineKind uint8

// Application definition.
//
// Ref to apdef.go for implementation
type IAppDef interface {
	IComment

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

	// Return GDoc by name.
	//
	// Returns nil if not found.
	GDoc(name QName) IGDoc

	// Return GRecord by name.
	//
	// Returns nil if not found.
	GRecord(name QName) IGRecord

	// Return CDoc by name.
	//
	// Returns nil if not found.
	CDoc(name QName) ICDoc

	// Return CRecord by name.
	//
	// Returns nil if not found.
	CRecord(name QName) ICRecord

	// Return WDoc by name.
	//
	// Returns nil if not found.
	WDoc(name QName) IWDoc

	// Return WRecord by name.
	//
	// Returns nil if not found.
	WRecord(name QName) IWRecord

	// Return ODoc by name.
	//
	// Returns nil if not found.
	ODoc(name QName) IODoc

	// Return ORecord by name.
	//
	// Returns nil if not found.
	ORecord(name QName) IORecord

	// Return Object by name.
	//
	// Returns nil if not found.
	Object(name QName) IObject

	// Return Element by name.
	//
	// Returns nil if not found.
	Element(name QName) IElement

	// Return View by name.
	//
	// Returns nil if not found.
	View(name QName) IView

	// Returns Command by name.
	//
	// Returns nil if not found.
	Command(QName) ICommand

	// Returns Query by name.
	//
	// Returns nil if not found.
	Query(QName) IQuery

	// Returns workspace by name.
	//
	// Returns nil if not found.
	Workspace(QName) IWorkspace
}

// Application definition builder
//
// Ref to appdef.go for implementation
type IAppDefBuilder interface {
	IAppDef
	ICommentBuilder

	// Adds new GDoc definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddGDoc(name QName) IGDocBuilder

	// Adds new GRecord definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddGRecord(name QName) IGRecordBuilder

	// Adds new CDoc definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddCDoc(name QName) ICDocBuilder

	// Adds new singleton CDoc definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddSingleton(name QName) ICDocBuilder

	// Adds new CRecord definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddCRecord(name QName) ICRecordBuilder

	// Adds new WDoc definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddWDoc(name QName) IWDocBuilder

	// Adds new WRecord definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddWRecord(name QName) IWRecordBuilder

	// Adds new ODoc definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddODoc(name QName) IODocBuilder

	// Adds new ORecord definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddORecord(name QName) IORecordBuilder

	// Adds new Object definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddObject(name QName) IObjectBuilder

	// Adds new Element definition with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddElement(name QName) IElementBuilder

	// Adds new definitions for view.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddView(QName) IViewBuilder

	// Adds new command.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddCommand(QName) ICommandBuilder

	// Adds new query.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddQuery(QName) IQueryBuilder

	// Adds new workspace.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if definition with name already exists.
	AddWorkspace(QName) IWorkspaceBuilder

	// Must be called after all definitions added. Validates and returns builded application definition or error
	Build() (IAppDef, error)
}

// Definition describes the entity, such as document, record or view. Definitions may have fields and containers.
//
// Ref to def.go for implementation
type IDef interface {
	// Parent cache
	App() IAppDef

	// Definition qualified name
	QName() QName

	// Definition kind.
	Kind() DefKind
}

// See [Issue #488](https://github.com/voedger/voedger/issues/488)
//
// Any definition may have comment
//
// Ref to commented.go for implementation
type IComment interface {
	// Returns comment
	Comment() string
}

type ICommentBuilder interface {
	// Sets comment as string with lines, concatenated with LF
	SetComment(...string)
}

// See [Issue #524](https://github.com/voedger/voedger/issues/524)
// Definition can be abstract:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element
//
// Ref to abstract.go for implementation
type IWithAbstract interface {
	// Returns is definition abstract
	Abstract() bool
}

type IWithAbstractBuilder interface {
	IWithAbstract

	// Makes definition abstract
	SetAbstract()
}

// Definitions with fields:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element,
//	- DefKind_ViewRecord_PartitionKey, DefKind_ViewRecord_ClusteringColumns and DefKind_ViewRecord_Value
//
// Ref. to field.go for implementation
type IFields interface {
	// Finds field by name.
	//
	// Returns nil if not found.
	Field(name string) IField

	// Returns fields count
	FieldCount() int

	// Enumerates all fields in add order.
	Fields(func(IField))

	// Finds reference field by name.
	//
	// Returns nil if field is not found, or field found but it is not a reference field
	RefField(name string) IRefField

	// Enumerates all reference fields. System field (sys.ParentID) is also enumerated
	RefFields(func(IRefField))

	// Returns reference fields count. System field (sys.ParentID) is also counted
	RefFieldCount() int

	// Enumerates all fields except system
	UserFields(func(IField))

	// Returns user fields count. System fields (sys.QName, sys.ID, …) do not count
	UserFieldCount() int
}

type IFieldsBuilder interface {
	IFields

	// Adds field specified name and kind.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if specified data kind is not allowed by definition kind.
	AddField(name string, kind DataKind, required bool, comment ...string) IFieldsBuilder

	// Adds reference field specified name and target refs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists.
	AddRefField(name string, required bool, ref ...QName) IFieldsBuilder

	// Adds string field specified name and restricts.
	//
	// If no restrictions specified, then field has maximum length 255.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if restricts are not compatible.
	AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder

	// Adds bytes field specified name and restricts.
	//
	// If no restrictions specified, then field has maximum length 255.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if restricts are not compatible.
	AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder

	// Adds verified field specified name and kind.
	//
	// # Panics:
	//   - if field name is empty,
	//   - if field name is invalid,
	//   - if field with name is already exists,
	//   - if data kind is not allowed by definition kind,
	//   - if no verification kinds are specified
	//
	//Deprecated: use SetVerifiedField instead
	AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IFieldsBuilder

	// Sets verification kind for specified field.
	//
	// If not verification kinds are specified then it means that field is not verifiable.
	//
	// # Panics:
	//   - if field not found.
	SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder
}

// Definitions with containers:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element,
//	- DefKind_ViewRecord and DefKind_ViewKey
//
// Ref. to container.go for implementation
type IContainers interface {
	// Finds container by name.
	//
	// Returns nil if not found.
	Container(name string) IContainer

	// Returns containers count
	ContainerCount() int

	// Enumerates all containers in add order.
	Containers(func(IContainer))
}

type IContainersBuilder interface {
	IContainers

	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if container with name already exists,
	//   - if definition name is empty,
	//   - if invalid occurrences,
	//   - if container definition kind is not compatible with parent definition kind.
	AddContainer(name string, def QName, min, max Occurs, comment ...string) IContainersBuilder
}

// Definitions with uniques:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord
//
// Ref. to unique.go for implementation
type IUniques interface {
	// Return unique by ID.
	//
	// Returns nil if not unique found
	UniqueByID(id UniqueID) IUnique

	// Return unique by name.
	//
	// Returns nil if not unique found
	UniqueByName(name string) IUnique

	// Return uniques count
	UniqueCount() int

	// Enumerates all uniques.
	Uniques(func(IUnique))

	// Returns single field unique.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	UniqueField() IField
}

type IUniquesBuilder interface {
	IUniques

	// Adds new unique with specified name and fields.
	// If name is omitted, then default name is used, e.g. `unique01`.
	//
	// # Panics:
	//   - if unique name is invalid,
	//   - if unique with name is already exists,
	//   - if definition kind is not supports uniques,
	//   - if fields list is empty,
	//   - if fields has duplicates,
	//   - if fields is already exists or overlaps with an existing unique,
	//   - if some field not found.
	AddUnique(name string, fields []string, comment ...string) IUniquesBuilder

	// Sets single field unique.
	// Calling SetUniqueField again changes unique field. If specified name is empty, then clears unique field.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	//
	// # Panics:
	//   - if field name is invalid,
	//   - if field not found,
	//   - if field is not required.
	SetUniqueField(name string) IUniquesBuilder
}

// Global document. DefKind() is DefKind_GDoc.
type IGDoc interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IGDocBuilder interface {
	IGDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Global document record. DefKind() is DefKind_GRecord.
//
// Ref. to gdoc.go for implementation
type IGRecord interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IGRecordBuilder interface {
	IGRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Configuration document. DefKind() is DefKind_CDoc.
type ICDoc interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract

	// Returns is singleton
	Singleton() bool
}

type ICDocBuilder interface {
	ICDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder

	// Sets CDoc singleton
	SetSingleton()
}

// Configuration document record. DefKind() is DefKind_CRecord.
//
// Ref. to cdoc.go for implementation
type ICRecord interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type ICRecordBuilder interface {
	ICRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Workflow document. DefKind() is DefKind_WDoc.
type IWDoc interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IWDocBuilder interface {
	IWDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Workflow document record. DefKind() is DefKind_WRecord.
//
// Ref. to wdoc.go for implementation
type IWRecord interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IWRecordBuilder interface {
	IWRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Operation document. DefKind() is DefKind_ODoc.
type IODoc interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IODocBuilder interface {
	IODoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// Operation document record. DefKind() is DefKind_ORecord.
//
// Ref. to odoc.go for implementation
type IORecord interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IORecordBuilder interface {
	IORecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// Object definition. DefKind() is DefKind_Object.
//
// Ref. to object.go for implementation
type IObject interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IObjectBuilder interface {
	IObject
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// Element definition. DefKind() is DefKind_Element.
//
// Ref. to object.go for implementation
type IElement interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IElementBuilder interface {
	IElement
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// View definition. DefKind() is DefKind_ViewRecord
//
// Ref to view.go for implementation
type IView interface {
	IDef
	IComment
	IContainers

	// Returns full (pk + ccols) view key definition
	Key() IViewKey

	// Returns view value definition
	Value() IViewValue
}

type IViewBuilder interface {
	IView
	ICommentBuilder

	// AddPartField adds specified field to view partition key definition. Fields is always required
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddPartField(name string, kind DataKind, comment ...string) IViewBuilder

	// AddClustColumn adds specified field to view clustering columns definition. Fields is optional
	//
	// # Panics:
	//	- if field already exists in partition key or value fields.
	AddClustColumn(name string, kind DataKind, comment ...string) IViewBuilder

	// AddValueField adds specified field to view value definition
	//
	// # Panics:
	//	- if field already exists in partition key or clustering columns fields.
	AddValueField(name string, kind DataKind, required bool, comment ...string) IViewBuilder
}

// View partition key definition. DefKind() is DefKind_ViewRecordPartitionKey
//
// Ref. to view.go for implementation
type IPartKey interface {
	IDef
	IComment
	IFields
}

// View clustering columns definition. DefKind() is DefKind_ViewRecordClusteringColumns
//
// Ref. to view.go for implementation
type IClustCols interface {
	IDef
	IComment
	IFields
}

// View full (pk + cc) key definition. DefKind() is DefKind_ViewRecordFullKey
//
// Partition key fields is required, clustering columns is not.
//
// Ref. to view.go for implementation
type IViewKey interface {
	IDef
	IComment
	IFields
	IContainers

	// Returns partition key definition
	Partition() IPartKey

	// Returns clustering columns definition
	ClustCols() IClustCols
}

// View value definition. DefKind() is DefKind_ViewRecord_Value
//
// Ref. to view.go for implementation
type IViewValue interface {
	IDef
	IComment
	IFields
}

// Describe single field.
//
// Ref to field.go for constants and implementation
type IField interface {
	IComment

	// Returns field name
	Name() string

	// Returns data kind for field
	DataKind() DataKind

	// Returns is field required
	Required() bool

	// Returns is field verifiable
	Verifiable() bool

	// Returns is field verifiable by specified verification kind
	VerificationKind(VerificationKind) bool

	// Returns is field has fixed width data kind
	IsFixedWidth() bool

	// Returns is field system
	IsSys() bool
}

// Reference field. Describe field with DataKind_RecordID.
//
// Use Refs() to obtain list of target references.
//
// Ref. to fields.go for implementation
type IRefField interface {
	IField

	// Returns list of target references
	Refs() []QName
}

// String field. Describe field with DataKind_string.
//
// Use Restricts() to obtain field restricts for length, pattern.
//
// Ref. to fields.go for implementation
type IStringField interface {
	IField

	// Returns restricts
	Restricts() IStringFieldRestricts
}

// String or bytes field restricts
type IStringFieldRestricts interface {
	// Returns minimum length
	//
	// Returns 0 if not assigned
	MinLen() uint16

	// Returns maximum length
	//
	// Returns DefaultFieldMaxLength (255) if not assigned
	MaxLen() uint16

	// Returns pattern regular expression.
	//
	// Returns nil if not assigned
	Pattern() *regexp.Regexp
}

// Bytes field. Describe field with DataKind_bytes.
//
// Use Restricts() to obtain field restricts for length.
//
// Ref. to fields.go for implementation
type IBytesField interface {
	IField

	// Returns restricts
	Restricts() IBytesFieldRestricts
}

type IBytesFieldRestricts = IStringFieldRestricts

// Field restrict. Describe single restrict for field.
//
// Interface functions to obtain new restricts:
//
// # String fields:
//   - MinLen(uint16)
//   - MaxLen(uint16)
//   - Pattern(string)
//
// Ref. to fields-restrict.go
type IFieldRestrict interface{}

// Describes single inclusion of child definition in parent definition.
//
// Ref to container.go for implementation
type IContainer interface {
	IComment

	// Returns name of container
	Name() string

	// Returns definition name of container
	QName() QName

	// Returns container definition.
	//
	// Returns nil if not found
	Def() IDef

	// Returns minimum occurs
	MinOccurs() Occurs

	// Returns maximum occurs
	MaxOccurs() Occurs

	// Returns is container system
	IsSys() bool
}

// Unique identifier type
type UniqueID uint32

// Describe single unique for definition.
//
// Ref to unique.go for implementation
type IUnique interface {
	IComment

	// returns parent definition
	Def() IDef

	// Returns name of unique.
	//
	// Name suitable for debugging or error messages. Unique identification provided by ID
	Name() string

	// Returns unique fields list. Fields are sorted alphabetically
	Fields() []IField

	// Unique identifier.
	//
	// Must be assigned during AppStruct construction by calling SetID(UniqueID)
	ID() UniqueID
}

// Entry point for extension
//
// Ref. to extension.go for implementation
type IExtension interface {
	IComment

	// Extension entry point name
	Name() string

	// Engine kind
	Engine() ExtensionEngineKind
}

// Command
//
// Ref. to command.go for implementation
type ICommand interface {
	IDef
	IComment

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Unlogged (secure) argument. Returns nil if not assigned
	UnloggedArg() IObject

	// Result. Returns nil if not assigned
	Result() IObject

	// Extension
	Extension() IExtension
}

type ICommandBuilder interface {
	ICommand
	ICommentBuilder

	// Sets command argument. Must be object or NullQName
	SetArg(QName) ICommandBuilder

	// Sets command unlogged (secure) argument. Must be object or NullQName
	SetUnloggedArg(QName) ICommandBuilder

	// Sets command result. Must be object or NullQName
	SetResult(QName) ICommandBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind, comment ...string) ICommandBuilder
}

// Query
//
// Ref. to query.go for implementation
type IQuery interface {
	IDef
	IComment

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Result. Returns nil if not assigned.
	//
	// If result is may be different, then NullQName is used
	Result() IObject

	// Extension
	Extension() IExtension
}

type IQueryBuilder interface {
	IQuery
	ICommentBuilder

	// Sets query argument. Must be object or NullQName
	SetArg(QName) IQueryBuilder

	// Sets query result. Must be object or NullQName
	SetResult(QName) IQueryBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind) IQueryBuilder
}

// Workspace
//
// Ref. to workspace.go for implementation
type IWorkspace interface {
	IDef
	IComment
	IWithAbstract

	// Returns definition by name.
	//
	// Nil is returned if not found
	Def(QName) IDef

	// Enumerates all workspace definitions
	Defs(func(IDef))

	// Workspace descriptor document.
	// See [#466](https://github.com/voedger/voedger/issues/466)
	//
	// Descriptor is CDoc document.
	// If the Descriptor is an abstract document, the workspace must also be abstract.
	Descriptor() QName
}

type IWorkspaceBuilder interface {
	IWorkspace
	ICommentBuilder
	IWithAbstractBuilder

	// Adds definition to workspace. Definition must be defined for application before.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	AddDef(QName) IWorkspaceBuilder

	// Sets descriptor.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	//	- if name is not CDoc
	SetDescriptor(QName) IWorkspaceBuilder
}
