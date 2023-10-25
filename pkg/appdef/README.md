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
        +Restricts() []IDataRestrict
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

    IGDoc --|> IDoc : inherits
    class IGDoc {
        <<interface>>
        +Kind()* TypeKind_GDoc
    }

    ICDoc --|> IDoc : inherits
    class ICDoc {
        <<interface>>
        +Kind()* TypeKind_CDoc
        +Singleton() bool
    }

    IWDoc --|> IDoc : inherits
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

    IElement --|> IStructure : inherits
    class IElement {
        <<interface>>
        +Kind()* TypeKind_Element
        +SystemField_Container() IField
    }

    IType <|-- IView : inherits
    class IView {
        <<interface>>
        +Kind()* TypeKind_ViewRecord
        +Key() IViewKey
        +Value() IViewValue
    }
            
    IType <|-- IFunction : inherits
    class IFunction {
        <<interface>>
        +Params() []IParam
        +Results() []IParam
    }

    IFunction <|-- ICommand : inherits
    class ICommand {
        <<interface>>
        +Kind()* TypeKind_Command
        +UnloggedParams() []IParam
    }

    IFunction <|-- IQuery : inherits
    class IQuery {
        <<interface>>
        +Kind()* TypeKind_Query
    }


    IWorkspace --|> IType : inherits
    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        +Abstract() bool
        +Types() []IType
    }
```

### Types composing

```mermaid
classDiagram


    class IData {
        <<interface>>
        +Kind()* TypeKind_Data
        +Ancestor() IData
        +Restricts() []IDataRestrict
    }
    IData "1" ..> "0..1" IData : ancestor ref

    class IArray {
        <<interface>>
        +Kind()* TypeKind_Array
        +MaxLen() uint
        +Elem() IType
    }

    class IGDoc {
        <<interface>>
        +Kind()* TypeKind_GDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IGDoc "1" o..> "0..*" IGRecord : contains

    class ICDoc {
        <<interface>>
        +Kind()* TypeKind_CDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
        +Singleton() bool
    }
    ICDoc "1" o..> "0..*" ICRecord : contains

    class IWDoc {
        <<interface>>
        +Kind()* TypeKind_WDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IWDoc "1" o..> "0..*" IWRecord : contains
                    
    class IODoc {
        <<interface>>
        +Kind()* TypeKind_ODoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IODoc "1" o..> "0..*" IODoc : contains
    IODoc "1" o..> "0..*" IORecord : contains

    class IGRecord {
        <<interface>>
        +Kind()* TypeKind_GRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IGRecord "1" o..> "0..*" IGRecord : contains

    class ICRecord {
        <<interface>>
        +Kind()* TypeKind_CRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    ICRecord "1" o..> "0..*" ICRecord : contains

    class IWRecord {
        <<interface>>
        +Kind()* TypeKind_WRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IWRecord "1" o..> "0..*" IWRecord : contains

    class IORecord {
        <<interface>>
        +Kind()* TypeKind_ORecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IORecord "1" o..> "0..*" IORecord : contains

    class IObject {
        <<interface>>
        +Kind()* TypeKind_Object
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IObject "1" o..> "0..*" IElement : contains

    class IElement {
        <<interface>>
        +Kind()* TypeKind_Element
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IElement "1" o..> "0..*" IElement : contains

    class IView {
        <<interface>>
        +Kind()* TypeKind_ViewRecord
        +Key() IViewKey
        +Value() IViewValue
    }
    IView "1" *--> "1" IViewKey : compose
    IView "1" *--> "1" IViewValue : compose
            
    class IViewKey {
        <<interface>>
        +Fields() []IField
        +PartKey() IViewPartKey
        +ClustCols() IViewClustCols
    }
    IViewKey "1" *--> "1" IViewPartKey : compose
    IViewKey "1" *--> "1" IViewClustCols : compose

    class IViewPartKey {
        <<interface>>
        +Fields() []IField
    }

    class IViewClustCols {
        <<interface>>
        +Fields() []IField
    }

    class IViewValue {
        <<interface>>
        +Fields() []IField
    }

    class ICommand {
        <<interface>>
        +Kind()* TypeKind_Command
        +Params() []IParam
        +UnloggedParams() []IParam
        +Results() []IParam
    }

    class IQuery {
        <<interface>>
        +Kind()* TypeKind_Query
        +Params() []IParam
        +Results() []IParam
    }

    class IWorkspace {
        <<interface>>
        +Kind()* TypeKind_Workspace
        +Abstract() bool
        +Types() []IType
    }

```

### Structures

Structured (documents, records, objects, elements) are those structural types that have fields and can contain containers with other structural types.

The inheritance and composing diagrams given below are expanded general diagrams of the types above.

### Structures inheritance

```mermaid
classDiagram
    direction BT
    namespace _ {
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
    }

    IRecord --|> IStructure : inherits

    IDoc --|> IRecord : inherits
    class IDoc {
        <<interface>>
    }

    IGDoc --|> IDoc : inherits
    class IGDoc {
        <<interface>>
        +Kind()* TypeKind_GDoc
    }

    ICDoc --|> IDoc : inherits
    class ICDoc {
        <<interface>>
        +Kind()* TypeKind_CDoc
        +Singleton() bool
    }

    IWDoc --|> IDoc : inherits
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
  IGDoc "1" o--> "0..*" IGRecord : has child

  class IGRecord {
    <<Interface>>
    IContainedRecord
  }
  IGRecord "1" o--> "0..*" IGRecord : has child

  class ICDoc {
    <<Interface>>
    IDoc
    +Singleton() bool
  }
  ICDoc "1" o--> "0..*" ICRecord : has child

  class ICRecord {
    <<Interface>>
    IContainedRecord
  }
  ICRecord "1" o--> "0..*" ICRecord : has child

  class IWDoc {
    <<Interface>>
    IDoc
  }
  IWDoc "1" o--> "0..*" IWRecord : has child

  class IWRecord {
    <<Interface>>
    IContainedRecord
  }
  IWRecord "1" o--> "0..*" IWRecord : has child

  class IODoc {
    <<Interface>>
    IDoc
  }
  IODoc "1" o--> "0..*" IORecord : has child

  class IORecord {
    <<Interface>>
    IContainedRecord
  }
  IORecord "1" o--> "0..*" IORecord : has child

  class IObject {
    <<Interface>>
    IStructure
  }
  IObject "1" o--> "0..*" IElement : has child

  class IElement {
    <<Interface>>
    IStructure
  }
  IElement "1" o--> "0..*" IElement : has child
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
    +VerificationKind(VerificationKind) bool
  }

  class IFields{
    <<Interface>>
    Field(string) IField
    FieldCount() int
    Fields(func(IField))
  }
  IFields "1" --* "0..*" IField : compose

  IFieldsBuilder --|> IFields : inherits
  class IFieldsBuilder {
    <<Interface>>
    AddField(…)
    AddVerifiedField(…)
    AddRefField(…)
    AddStringField(…)
  }

  IRefField --|> IField : inherits
  class IRefField {
    <<Interface>>
    Refs() []QName
  }

  IStringField --|> IField : inherits
  class IStringField {
    <<Interface>>
    Refs() []QName
    Restricts() IStringFieldRestricts
  }

  IStringField "1" --o "1" IStringFieldRestricts : aggregates
  class IStringFieldRestricts {
    <<Interface>>
    MinValue() uint16
    MaxValue() uint16
    Pattern() string
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
    ContainerDef(string) IDef
    Containers(func(IContainer))
  }
  IContainers "1" --* "0..*" IContainer : compose

  IContainersBuilder --|> IContainers : inherits
  class IContainersBuilder {
    <<Interface>>
    AddContainer(…) IContainer
  }

  class IUnique {
    <<Interface>>
    +Name() string
    +ID() UniqueID
    +Fields() []IFeld
  }

  class IUniques{
    <<Interface>>
    UniqueByID(UniqueID) IUnique
    UniqueByName(string) IUnique
    UniqueCount() int
    Uniques(func(IUnique))
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
  IView "1" *--> "1" IViewKey : has
  IView "1" *--> "1" IViewValue : has

  class IViewKey {
    <<Interface>>
    IFields
    +PartKey() IViewPartKey
    +ClustCols() IViewClustCols
  }
  IViewKey "1" *--> "1..*" IField : has
  IViewKey "1" *--> "1" IViewPartKey : has
  IViewKey "1" *--> "1" IViewClustCols : has

  class IViewPartKey {
    <<Interface>>
    IFields
  }
  IViewPartKey "1" *--> "1..*" IField : has

  class IViewClustCols {
    <<Interface>>
    IFields
  }
  IViewClustCols "1" *--> "1..*" IField : has

  class IViewValue {
    <<Interface>>
    IFields
  }
  IViewValue "1" *--> "1..*" IField : has

  class IField {
    <<interface>>
    …
  }
```

### Functions, commands and queries

```mermaid
classDiagram
    
    direction TB

    class IType {
        <<interface>>
        +QName() QName
        +Kind()* TypeKind
        +Comment() []string
    }

    IType <|-- IFunction : inherits
    class IFunction {
        <<interface>>
        +Extension() IExtension
        +Param() IObject
        +Result() IObject
    }

    IFunction <|-- ICommand : inherits
    class ICommand {
        <<interface>>
        +Kind()* TypeKind_Command
        +UnloggedParam() IObject
    }

    IFunction <|-- IQuery : inherits
    class IQuery {
        <<interface>>
        +Kind()* TypeKind_Query
    }

    IFunction "1" *-- "1" IExtension : has
    class IExtension {
        <<interface>>
        +Name() string
        +Engine() ExtensionEngineKind
        +Comment() []string
    }

    IExtension "1" ..> "1" ExtensionEngineKind : refs
    class ExtensionEngineKind {
        <<enumeration>>
        BuiltIn
        WASM
    }
```

*Rem*: In the above diagram the Param and Result of the function are `IObject`, in future versions it will be changed to an array of `[]IParam` and renamed to plural (`Params`, `Results`).

## Restrictions

### Names

- Only letters (from `A` to `Z` and from `a` to `z`), digits (from `0` to `9`) and underscore symbol (`_`) are used.
- First symbol must be letter or underscore.
- Maximum length of name is 255.

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
- Maximum string field length is 1024.

### Containers

- Maximum containers per structure is 65536.

### Uniques

- Maximum fields per unique is 256
- Maximum uniques per structure is 100.
