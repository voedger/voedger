# Application Definition

[![codecov](https://codecov.io/gh/voedger/voedger/appdef/branch/main/graph/badge.svg?token=u6VrbqKtnn)](https://codecov.io/gh/voedger/voedger/appdef)

## Types

### Types inheritance

```mermaid
classDiagram
    class IAppDef {
        <<interface>>
        +Type(QName) IType
        +Data(QName) IData
        +GDoc(QName) IGDoc
        +CDoc(QName) ICDoc
        +WDoc(QName) IWDoc
        +ODoc(QName) IODoc
        +View(QName) IView
        +Command(QName) ICommand
        +Query(QName) IQuery
        +Role(QName) IRole
        +Projector(QName) IProjector
        +Job(QName) IJob
        +Workspace(QName) IWorkspace
    }
    IAppDef "1" *--> "0..*" IType : compose

    class IType {
        <<interface>>
        +QName() QName
        +Kind()* TypeKind
        +Comment() []string
    }

    IData --|> IType : inherits
    class IData {
        <<interface>>
        +Kind()* TypeKind_Data
        +Ancestor() IData
        +Constraints() []IConstraint
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
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
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

    IWorkspace --|> IType : inherits
    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        +Abstract() bool
        +Descriptor() QName
        +Types() []IType
    }

    IRole --|> IType : inherits
    class IRole {
        <<interface>>
        +Kind()* TypeKind_Role
        +Privileges() []IPrivilege
    }
```

### Data types

```mermaid
classDiagram
    direction BT

  class IAppDef {
    <<Interface>>
    +DataTypes(inclSys bool) []IData
    +SysData(DataKind) IData
  }

  IData "0..*" <--o "1" IAppDef : DataTypes
  IData "1..DataKind_count" <--o "1" IAppDef : SysData

    class IType {
        <<interface>>
        +Name() QName
        +Kind() TypeKind
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
    note for IConstraint " - minLen() uint
                           - maxLen() uint
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
%%    namespace _ {
        class IStructure {
            <<interface>>
            +Abstract() bool
            +Fields() []IField
            +Containers() []IContainer
            +Uniques() []IUnique
            +SystemField_QName() IField
        }

        class IRecord {
            <<interface>>
            +SystemField_ID() IField
            +SystemField_IsActive() IField
        }
%%    }

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

  class IAppDef {
    <<Interface>>
    +Structures() []IStructure
    +Records() []IRecord
    +Singletons() []ISingleton
    +GDocs()[]IGDoc
    +GRecords()[]IGRecord
    +CDocs()[]ICDoc
    +CRecords()[]ICRecord
    +WDocs()[]IWDoc
    +WRecords()[]IWRecord
    +ODocs()[]IODoc
    +ORecords()[]IORecord
    +Objects()[]IObject
  }
  IAppDef "1" o--> "0..*" IStructure : Structures
  IAppDef "1" o--> "0..*" IRecord : Records
  IAppDef "1" o--> "0..*" ISingleton : Singletons
  IAppDef "1" o--> "0..*" IGDoc : GDocs
  IAppDef "1" o--> "0..*" IGRecord : GRecords
  IAppDef "1" o--> "0..*" ICDoc : CDocs
  IAppDef "1" o--> "0..*" ICRecord : CRecords
  IAppDef "1" o--> "0..*" IWDoc : WDocs
  IAppDef "1" o--> "0..*" IWRecord : WRecords
  IAppDef "1" o--> "0..*" IODoc : ODocs
  IAppDef "1" o--> "0..*" IORecord : ORecords
  IAppDef "1" o--> "0..*" IObject : Objects

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
    +Name() FieldName
    +DataKind() DataKind
    +Required() bool
    +Verified() bool
    +VerificationKind() []VerificationKind
    +Constraints() []IConstraint
  }

  class IFields{
    <<Interface>>
    Field(FieldName) IField
    FieldCount() int
    Fields() []IField
  }
  IFields "1" --* "0..*" IField : compose

  IFieldsBuilder --|> IFields : inherits
  class IFieldsBuilder {
    <<Interface>>
    AddField(…)
    AddVerifiedField(…)
    AddRefField(…)
    AddStringField(…)
    AddConstraints(IConstraint...)
  }

  IRefField --|> IField : inherits
  class IRefField {
    <<Interface>>
    Refs() []QName
  }

  class IContainer {
    <<Interface>>
    +Name() string
    +Def() IDef
    +MinOccurs() int
    +MaxOccurs() int
  }

  class IContainers{
    <<Interface>>
    Container(string) IContainer
    ContainerCount() int
    ContainerDef() [string]IType
    Containers() []IContainer
  }
  IContainers "1" --* "0..*" IContainer : compose

  IContainersBuilder --|> IContainers : inherits
  class IContainersBuilder {
    <<Interface>>
    AddContainer(…) IContainer
  }

  class IUnique {
    <<Interface>>
    +Name() QName
    +Fields() []IFeld
  }

  class IUniques{
    <<Interface>>
    UniqueByName(QName) IUnique
    UniqueCount() int
    Uniques() []IUnique
  }
  IUniques "1" --* "0..*" IUnique : compose

  IUniquesBuilder --|> IUniques : inherits
  class IUniquesBuilder {
    <<Interface>>
    AddUnique(…) IUnique
  }
```

### Views

```mermaid
classDiagram
  class IType{
    <<Interface>>
    +Kind()* TypeKind
    +QName() QName
  }

  IType <|-- IView : inherits
  class IView {
    <<Interface>>
    +Kind()* TypeKind_View
    IFields
    +Key() IViewKey
    +Value() IViewValue
  }
  IView "1" *--> "1" IViewKey : Key
  IView "1" *--> "1" IViewValue : Value

  class IViewKey {
    <<Interface>>
    IFields
    +PartKey() IViewPartKey
    +ClustCols() IViewClustCols
  }
  IViewKey "1" *--> "1..*" IField : fields
  IViewKey "1" *--> "1" IViewPartKey : PartKey
  IViewKey "1" *--> "1" IViewClustCols : ClustCols

  class IViewPartKey {
    <<Interface>>
    IFields
  }
  IViewPartKey "1" *--> "1..*" IField : fields

  class IViewClustCols {
    <<Interface>>
    IFields
  }
  IViewClustCols "1" *--> "1..*" IField : fields

  class IViewValue {
    <<Interface>>
    IFields
  }
  IViewValue "1" *--> "1..*" IField : fields

  class IField {
    <<interface>>
    …
  }

    class IAppDef {
      …
      +Views() []IView
    }

    IAppDef "1" *--> "0..*" IView : Views
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
        +Enum(func(IStorage))
        +Len() int
        +Map() map[QName] []QName
        +Storage(QName) IStorage
    }
    IStorages "1" *--> "0..*" IStorage : Storages
    class IStorage {
        <<interface>>
        +Comment() : []string
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
        +Events() IProjectorEvents
    }

    IProjector "1" *--> "1" IProjectorEvents : Events
    IProjectorEvents "1" *--> "1..*" IProjectorEvent : Event
    class IProjectorEvents {
        <<interface>>
        +Enum(func(IProjectorEvent))
        +Event(QName) IProjectorEvent
        +Len() int
        +Map() map[QName] []ProjectorEventKind
    }
    class IProjectorEvent {
        <<interface>>
        +Comment() []string
        +On() IType
        +Kind() []ProjectorEventKind
    }

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

    class IAppDef {
      …
      +Extensions() []IExtension
      +Functions() []IFunction
      +Commands() []ICommand
      +Queries() []IQuery
      +Projectors() []IProjector
      +Jobs() []IJob
    }

    IAppDef "1" *--> "0..*" IExtension : Extensions
    IAppDef "1" *--> "0..*" IFunction : Functions
    IAppDef "1" *--> "0..*" ICommand : Commands
    IAppDef "1" *--> "0..*" IQuery : Queries
    IAppDef "1" *--> "0..*" IProjector : Projectors
    IAppDef "1" *--> "0..*" IJob : Jobs
```

*Rem*: In the above diagram the Param and Result of the function are `IType`, in future versions it will be changed to an array of `[]IParam` and renamed to plural (`Params`, `Results`).

### Workspaces

### Roles and privileges

```mermaid
    classDiagram
    IType <|-- IRole : inherits
    class IRole {
        <<interface>>
        +Kind()* TypeKind_Role
        +Privileges() []IPrivilege
    }

    IRole "1" *--> "1..*" IPrivilege : On

    class IPrivilege {
        <<interface>>
        +Comment() []string
        +Kinds() []PrivilegeKind
        +IsGranted() bool
        +IsRevoked() bool
        +On() QNames
        +Fields() []FieldName
    }

    IPrivilege "1" *--> "1..*" PrivilegeKind : Kinds
    
    class PrivilegeKind {
        <<enumeration>>
        Insert
        Update
        Select
        Execute
        Inherits
    }

    IPrivilege "1" *--> "1..*" QName : On
    note for QName "types on which the privilege is granted or revoked"

    class IAppDef {
      …
      +Roles() []IRole
      +Privileges() []IPrivilege
    }

    IAppDef "1" *--> "0..*" IRole : Roles
    IAppDef "1" *--> "0..*" IPrivilege : all application privileges
```

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
