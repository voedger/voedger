# Application Definition

## Types

### Types inheritance

```mermaid
classDiagram
    class IType {
        <<interface>>
        +QName() QName
        +Kind()* TypeKind
        +Comment() string
        +Tags() []ITag
    }

    ITag --|> IType : inherits
    class ITag {
        <<interface>>
        +Kind()* TypeKind_Tag
    }

    IData --|> IType : inherits
    class IData {
        <<interface>>
        +Kind()* TypeKind_Data
        +Ancestor() IData
        +Constraints() map[ConstraintKind]IConstraint
    }

    IArray --|> IType : inherits
    class IArray {
        <<interface>>
        +Kind()* TypeKind_Array
        +MaxLen() uint
        +Elem() IType
    }

    IType <|-- IStructure : inherits
    class IStructure {
        <<interface>>
        IWithFields
        IWithContainers
        IWithUniques
        +Abstract() bool
        +SystemField_QName() IField
    }

    IStructure <|-- IRecord  : inherits
    class IRecord {
        <<interface>>
        +SystemField_ID() IField
        +SystemField_IsActive() IField
    }

    IDoc --|> IRecord : inherits
    class IDoc {
        <<interface>>
    }

    ISingleton --|> IDoc : inherits
    class ISingleton {
        <<interface>>
        +Singleton() bool
    }

    IGDoc --|> IDoc : inherits
    class IGDoc {
        <<interface>>
        +Kind()* TypeKind_GDoc
    }

    ICDoc --|> ISingleton : inherits
    class ICDoc {
        <<interface>>
        +Kind()* TypeKind_CDoc
    }

    IWDoc --|> ISingleton : inherits
    class IWDoc {
        <<interface>>
        +Kind()* TypeKind_WDoc
    }

    IODoc --|> IDoc: inherits
    class IODoc {
        <<interface>>
        +Kind()* TypeKind_ODoc
    }

    IRecord <|-- IContainedRecord  : inherits
    class IContainedRecord {
        <<interface>>
        +SystemField_ParentID() IField
        +SystemField_Container() IField
    }

    IContainedRecord <|-- IGRecord : inherits
    class IGRecord {
        <<interface>>
        +Kind()* TypeKind_GRecord
    }

    IContainedRecord <|-- ICRecord : inherits
    class ICRecord {
        <<interface>>
        +Kind()* TypeKind_CRecord
    }

    IContainedRecord <|-- IWRecord : inherits
    class IWRecord {
        <<interface>>
        +Kind()* TypeKind_WRecord
    }

    IContainedRecord <|-- IORecord : inherits
    class IORecord {
        <<interface>>
        +Kind()* TypeKind_ORecord
    }

    IObject --|> IStructure : inherits
    class IObject {
        <<interface>>
        +Kind()* TypeKind_Object
    }

    IType <|-- IView : inherits
    class IView {
        <<interface>>
        +Kind()* TypeKind_ViewRecord
        +Key() IViewKey
        +Value() IViewValue
    }

    IType <|-- IExtension : inherits
    class IExtension {
        <<interface>>
        +Name() string
        +Engine() ExtensionEngineKind
        +States() IStorages
        +Intents() IStorages
    }

    IExtension <|-- IFunction : inherits
    class IFunction {
        <<interface>>
        +Param() IType
        +Result() IType
    }

    IFunction <|-- ICommand : inherits
    class ICommand {
        <<interface>>
        +Kind()* TypeKind_Command
        +UnloggedParam() IType
    }

    IFunction <|-- IQuery : inherits
    class IQuery {
        <<interface>>
        +Kind()* TypeKind_Query
    }

    IExtension <|-- IProjector : inherits
    class IProjector {
        <<interface>>
        +Kind()* TypeKind_Projector
        +WantErrors() bool
        +Events() IProjectorEvents
    }

    IExtension <|-- IJob : inherits
    class IJob {
        <<interface>>
        +Kind()* TypeKind_Job
        +CronSchedule() string
    }

    IRole --|> IType : inherits
    class IRole {
        <<interface>>
        +Kind()* TypeKind_Role
        +Published() bool
    }

    IRate --|> IType : inherits
    class IRate {
        <<interface>>
        +Kind()* TypeKind_Rate
    }

    ILimit --|> IType : inherits
    class ILimit {
        <<interface>>
        +Kind()* TypeKind_Limit
    }

    IWorkspace --|> IType : inherits
    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        +Types()[]IType
        ...
    }
```

### Types iterators

```mermaid
classDiagram

  class IAppDef {
    <<interface>>
    +Types() []IType
    +Workspaces() []IWorkspace
    ...
  }
  IAppDef "1" *--> "1" iterator : Types

  class IWorkspace {
    <<interface>>
    +LocalTypes()[]IType
    +Types()[]IType
    …
  }
  IWorkspace "1" *--> "2" iterator : LocalTypes and Types

  class iterator {
    <<func([]IType)iter.Seq[T]>>

    +CDocs([]IType) iter.Seq[ICDoc]
    +Commands([]IType) iter.Seq[ICommand]
    +CRecords([]IType) iter.Seq[ICRecord]
    +DataTypes([]IType) iter.Seq[IData]
    +Extensions([]IType) iter.Seq[IExtension]
    +Functions([]IType) iter.Seq[IFunction]
    +GDocs([]IType) iter.Seq[IGDoc]
    +GRecords([]IType) iter.Seq[IGRecord]
    +Jobs([]IType) iter.Seq[IJob]
    +Limits([]IType) iter.Seq[ILimit]
    +Objects([]IType) iter.Seq[IObject]
    +ODocs([]IType) iter.Seq[IODoc]
    +ORecords([]IType) iter.Seq[IORecord]
    +Projectors([]IType) iter.Seq[IProjector]
    +Queries([]IType) iter.Seq[IQuery]
    +Rates([]IType) iter.Seq[IRate]
    +Records([]IType) iter.Seq[IRecord]
    +Roles([]IType) iter.Seq[IRole]
    +Singletons([]IType) iter.Seq[ISingleton]
    +Structures([]IType) iter.Seq[IStructure]
    +Tags([]IType) iter.Seq[ITag]
    +Views([]IType) iter.Seq[IView]
    +WDocs([]IType) iter.Seq[IWDoc]
    +WRecords([]IType) iter.Seq[IWRecord]

    +TypesByKind([]IType,TypeKind) []T
    +TypesByKinds([]IType,TypeKinds) iter.Seq[T]
  }

```

### Data types

```mermaid
classDiagram
    direction BT

    class IType {
        <<interface>>
        +Comment() string
        +Name() QName
        +Kind() TypeKind
        ...
    }

    IData --|> IType : inherits
    class IData {
        <<interface>>
        +Name()* QName
        +Kind()* TypeKind_Data
        +DataKind() DataKind
        +Ancestor() IData
        +Constraints() []IConstraint
    }

    Name "1" <--* "1" IData : Name
    class Name {
        <<QName>>
    }
    note for Name "- for built-in types sys.int32, sys.float64, etc.,
                   - for custom types — user-defined and
                   - NullQName for anonymous types"

    DataKind "1" <--* "1" IData : Kind
    class DataKind {
        <<DataKind>>
    }
    note for DataKind " - null
                        - int8
                        - int16
                        - int32
                        - int64
                        - float32
                        - float64
                        - bytes
                        - string
                        - QName
                        - bool
                        - RecordID
                        - Record
                        - Event"

    Ancestor "1" <--* "1" IData : Ancestor
    class Ancestor {
        <<IData>>
    }
    note for Ancestor "  - data type from which the user data type is inherits or
                         - nil for built-in types"

    IConstraint "0..*" <--*  "1" IData : Constraints
    class IConstraint {
        <<interface>>
        +Kind() ConstraintKind
        +Value() any
    }
    note for IConstraint " - MinLen() uint
                           - MaxLen() uint
                           - Pattern() RegExp
                           - MinInclusive() float
                           - MinExclusive() float
                           - MaxInclusive() float
                           - MaxExclusive() float
                           - Enum() []enumerable"
```

### Structures

Structured (documents, records, objects) are those structural types that have fields and can contain containers with other structural types.

The inheritance and composing diagrams given below are expanded general diagrams of the types above.

### Structures inheritance

```mermaid
classDiagram
    direction BT
    namespace _ {
        class IStructure {
            <<interface>>
            IWithFields
            IWithContainers
            IWithUniques
            +Abstract() bool
            +SystemField_QName() IField
        }

        class IRecord {
            <<interface>>
            +SystemField_ID() IField
            +SystemField_IsActive() IField
        }
    }

    IRecord --|> IStructure : inherits

    IDoc --|> IRecord : inherits
    class IDoc {
        <<interface>>
    }

    ISingleton --|> IDoc : inherits
    class ISingleton {
        <<interface>>
        +Singleton() bool
    }

    IGDoc --|> IDoc : inherits
    class IGDoc {
        <<interface>>
        +Kind()* TypeKind_GDoc
    }

    ICDoc --|> ISingleton : inherits
    class ICDoc {
        <<interface>>
        +Kind()* TypeKind_CDoc
    }

    IWDoc --|> ISingleton : inherits
    class IWDoc {
        <<interface>>
        +Kind()* TypeKind_WDoc
    }

    IODoc --|> IDoc: inherits
    class IODoc {
        <<interface>>
        +Kind()* TypeKind_ODoc
    }

    IRecord <|-- IContainedRecord  : inherits
    class IContainedRecord {
        <<interface>>
        +SystemField_ParentID() IField
        +SystemField_Container() IField
    }

    IContainedRecord <|-- IGRecord : inherits
    class IGRecord {
        <<interface>>
        +Kind()* TypeKind_GRecord
    }

    IContainedRecord <|-- ICRecord : inherits
    class ICRecord {
        <<interface>>
        +Kind()* TypeKind_CRecord
    }

    IContainedRecord <|-- IWRecord : inherits
    class IWRecord {
        <<interface>>
        +Kind()* TypeKind_WRecord
    }

    IContainedRecord <|-- IORecord : inherits
    class IORecord {
        <<interface>>
        +Kind()* TypeKind_ORecord
    }
```

### Structures composing

```mermaid
classDiagram
  direction TB

  IGDoc "1" o--> "0..*" IGRecord : children
  IGRecord "1" o--> "0..*" IGRecord : children

  ICDoc "1" o--> "0..*" ICRecord : children
  ICRecord "1" o--> "0..*" ICRecord : children

  IWDoc "1" o--> "0..*" IWRecord : children
  IWRecord "1" o--> "0..*" IWRecord : children

  IODoc "1" o--> "0..*" IORecord : children
  IODoc "1" o--> "0..*" IODoc : children document
  IORecord "1" o--> "0..*" IORecord : children

  IObject "1" o--> "0..*" IObject : children
```

### Fields, Containers, Uniques

```mermaid
classDiagram

  class IField {
    <<Interface>>
    +Comment() string
    +Name() FieldName
    +DataKind() DataKind
    +Required() bool
    +Verified() bool
    +VerificationKind() []VerificationKind]
    +Constraints() map[ConstraintKind]IConstraint
  }

  class IWithFields{
    <<Interface>>
    Field(FieldName) IField
    FieldCount() int
    Fields() []IField
  }
  IWithFields "1" --* "0..*" IField : compose

  IRefField --|> IField : inherits
  class IRefField {
    <<Interface>>
    Refs() []QName
  }

  class IContainer {
    <<Interface>>
    +Name() string
    +Type() IType
    +MinOccurs() int
    +MaxOccurs() int
  }

  class IWithContainers{
    <<Interface>>
    Container(string) IContainer
    ContainerCount() int
    Containers() []IContainer
  }
  IWithContainers "1" --* "0..*" IContainer : compose

  class IUnique {
    <<Interface>>
    +Name() QName
    +Fields() []IField
  }

  class IWithUniques{
    <<Interface>>
    UniqueByName(QName) IUnique
    UniqueCount() int
    Uniques() map[QName] IUnique
  }
  IWithUniques "1" --* "0..*" IUnique : compose
```

### Views

```mermaid
classDiagram
  class IType{
    <<Interface>>
    +Kind()* TypeKind
    +QName() QName
    ...
  }

  IType <|-- IView : inherits
  class IView {
    <<Interface>>
    +Kind()* TypeKind_View
    IWithFields
    +Key() IViewKey
    +Value() IViewValue
  }
  IView "1" *--> "1" IViewKey : Key
  IView "1" *--> "1" IViewValue : Value

  class IViewKey {
    <<Interface>>
    IWithFields
    +PartKey() IViewPartKey
    +ClustCols() IViewClustCols
  }
  IViewKey "1" *--> "1" IViewPartKey : PartKey
  IViewKey "1" *--> "1" IViewClustCols : ClustCols

  class IViewPartKey {
    <<Interface>>
    IWithFields
  }

  class IViewClustCols {
    <<Interface>>
    IWithFields
  }

  class IViewValue {
    <<Interface>>
    IWithFields
  }
```

### Extensions

```mermaid
    classDiagram
    IType <|-- IExtension : inherits
    class IExtension {
        <<interface>>
        +Name() string
        +Engine() ExtensionEngineKind
        +States() IStorages
        +Intents() IStorages
    }

    IExtension "1" ..> "1" ExtensionEngineKind : Engine
    class ExtensionEngineKind {
        <<enumeration>>
        BuiltIn
        WASM
    }

    IExtension "1" *--> "1" IStorages : States
    IExtension "1" *--> "1" IStorages : Intents
    class IStorages {
        <<interface>>
        +Names()[]QName
        +Storage(QName) IStorage
    }
    IStorages "1" *--> "0..*" IStorage : Storages
    class IStorage {
        <<interface>>
        +Comment() : string
        +Name(): QName
        +QNames() []QName
    }

    IExtension <|-- IFunction : inherits
    class IFunction {
        <<interface>>
        +Param() IType
        +Result() IType
    }

    IFunction <|-- ICommand : inherits
    class ICommand {
        <<interface>>
        +Kind()* TypeKind_Command
        +UnloggedParam() IType
    }

    IFunction <|-- IQuery : inherits
    class IQuery {
        <<interface>>
        +Kind()* TypeKind_Query
    }

    IExtension <|-- IProjector : inherits
    class IProjector {
        <<interface>>
        +Kind()* TypeKind_Projector
        +WantErrors() bool
        +Events()[]IProjectorEvent
    }

    IProjector "1" *--> "1..*" IProjectorEvent : events
    class IProjectorEvent {
        <<interface>>
        +Comment() string
        +Ops() []OperationKind]
        +Filter() IFilter
    }

    IProjectorEvent "1" *--> "1" IFilter : filter

    note for IFilter "see filter/readme.md"

    IProjectorEvent "1" ..> "1..*" ProjectorEventKind : Kind
    class ProjectorEventKind {
        <<enumeration>>
        Insert
        Update
        Activate
        Deactivate
        Execute
        ExecuteWithParam
    }

    IExtension <|-- IJob : inherits
    class IJob {
        <<interface>>
        +Kind()* TypeKind_Job
        +CronSchedule() string
    }
```

*Rem*: In the above diagram the Param and Result of the function are `IType`, in future versions it will be changed to an array of `[]IParam` and renamed to plural (`Params`, `Results`).

### Roles and ACL

```mermaid
    classDiagram
    IType <|-- IWorkspace : inherits

    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        ...
        +ACL() []IACLRule
    }
    IWorkspace "1" *--> "0..*" IACLRule : ACL for workspace
    IWorkspace "1" *--> "0..*" IRole : roles

    IType <|-- IRole : inherits
    class IRole {
        <<interface>>
        +Kind()* TypeKind_Role
        +Published() bool
    }

    class IACLRule {
        <<interface>>
        +Comment() string
        +Ops() []OperationKind]
        +Policy() PolicyKind
        +Filter() IACLFilter
        +Principal() IRole
        +Workspace() IWorkspace
    }

    IACLRule "1" *--> "1..*" OperationKind : operations

    class OperationKind {
        <<enumeration>>
        Insert
        Update
        Select
        Execute
        Inherits
    }

    IACLRule "1" *--> "1" PolicyKind : policy

    class PolicyKind {
        <<enumeration>>
        Allow
        Deny
    }

    IACLRule "1" *--> "1" IACLFilter : resources filter

    class IACLFilter {
        <<interface>>
        +Fields()[]FieldName
    }

    IFilter <|-- IACLFilter : inherits

    note for IFilter "see filter/readme.md"

    IACLRule "1" *--> "1" IRole : principal
```

### Filters

### Tags

### Rates and Limits

### Workspaces

## Restrictions

### Names

- Only letters (from `A` to `Z` and from `a` to `z`), digits (from `0` to `9`) and underscore symbol (`_`) are used.
- First symbol must be letter or underscore.
- Maximum length of name is 255.
- Names are case sensitive.
- System level names can contains buck char (`$`).

Valid names examples:

```text
  Foo
  bar
  FooBar
  foo_bar
  f007
  _f00_bar
```

Invalid names examples:

```text
  Fo-o
  7bar
```

### Fields

- Maximum fields per structure is 65536.
- Maximum string and bytes field length is 65535.

### Containers

- Maximum containers per structure is 65536.

### Uniques

- Maximum fields per unique is 256
- Maximum uniques per structure is 100.

### Singletons

- Maximum singletons per application is 512.
