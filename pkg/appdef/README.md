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
        +Projector(QName) IProjector
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
        +Extension() IExtension
        +WantErrors() bool
        +Events() IProjectorEvents
        +States() IStorages
        +Intents() IStorages
    }

    IWorkspace --|> IType : inherits
    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        +Abstract() bool
        +Descriptor() QName
        +Types() []IType
    }
```

### Data types

```mermaid
classDiagram
    direction BT
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

  class IGDoc {
    <<Interface>>
    IDoc
  }
  IGDoc "1" o--> "0..*" IGRecord : children

  class IGRecord {
    <<Interface>>
    IContainedRecord
  }
  IGRecord "1" o--> "0..*" IGRecord : children

  class ICDoc {
    <<Interface>>
    ISingleton
  }
  ICDoc "1" o--> "0..*" ICRecord : children

  class ICRecord {
    <<Interface>>
    IContainedRecord
  }
  ICRecord "1" o--> "0..*" ICRecord : children

  class IWDoc {
    <<Interface>>
    ISingleton
  }
  IWDoc "1" o--> "0..*" IWRecord : children

  class IWRecord {
    <<Interface>>
    IContainedRecord
  }
  IWRecord "1" o--> "0..*" IWRecord : children

  class IODoc {
    <<Interface>>
    IDoc
  }
  IODoc "1" o--> "0..*" IORecord : children

  class IORecord {
    <<Interface>>
    IContainedRecord
  }
  IORecord "1" o--> "0..*" IORecord : children

  class IObject {
    <<Interface>>
    IStructure
  }
  IObject "1" o--> "0..*" IObject : children
```

### Fields, Containers, Uniques

```mermaid
classDiagram

  class IField {
    <<Interface>>
    +Name() string
    +DataKind() DataKind
    +Required() bool
    +Verified() bool
    +VerificationKind() []VerificationKind
    +Constraints() []IConstraint
  }

  class IFields{
    <<Interface>>
    Field(string) IField
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
```

### Functions, commands, queries and projectors

```mermaid
    classDiagram
    IType <|-- IExtension : inherits
    class IExtension {
        <<interface>>
        +Name() string
        +Engine() ExtensionEngineKind
    }

    IExtension "1" ..> "1" ExtensionEngineKind : Engine
    class ExtensionEngineKind {
        <<enumeration>>
        BuiltIn
        WASM
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
        +Extension() IExtension
        +WantErrors() bool
        +Events() IProjectorEvents
        +States() IStorages
        +Intents() IStorages
    }

    IProjector "1" *--> "1" IProjectorEvents : Events
    IProjectorEvents "1" *--> "1..*" IProjectorEvent : Events
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

    IProjector "1" *--> "1" IStorages : States
    IProjector "1" *--> "1" IStorages : Intents
    IStorages "1" *--> "0..*" IStorage : Storages
    class IStorage {
        QName
        +Comment() : []string
        +QNames() []QName
    }
```

*Rem*: In the above diagram the Param and Result of the function are `IType`, in future versions it will be changed to an array of `[]IParam` and renamed to plural (`Params`, `Results`).

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