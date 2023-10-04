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
        +Kind() TypeKind
        +Comment() []string
    }

    IData --|> IType : inherits
    class IData {
        <<interface>>
        ~Kind => TypeKind_Data
        +Ancestor() IData
        +Restricts() []IDataRestrict
    }

    IArray --|> IType : inherits
    class IArray {
        <<interface>>
        ~Kind => TypeKind_Array
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
    }

    IDoc --|> IStructure : inherits
    class IDoc {
        <<interface>>
    }

    IGDoc --|> IDoc : inherits
    class IGDoc {
        <<interface>>
        ~Kind => TypeKind_GDoc
    }

    ICDoc --|> IDoc : inherits
    class ICDoc {
        <<interface>>
        ~Kind => TypeKind_CDoc
        +Singleton() bool
    }

    IWDoc --|> IDoc : inherits
    class IWDoc {
        <<interface>>
        ~Kind => TypeKind_WDoc
    }
                    
    IODoc --|> IDoc: inherits
    class IODoc {
        <<interface>>
        ~Kind => TypeKind_ODoc
    }

    IStructure <|-- IRecord  : inherits
    class IRecord {
        <<interface>>
    }

    IRecord <|-- IGRecord : inherits
    class IGRecord {
        <<interface>>
        ~Kind => TypeKind_GRecord
    }

    IRecord <|-- ICRecord : inherits
    class ICRecord {
        <<interface>>
        ~Kind => TypeKind_CRecord
    }

    IRecord <|-- IWRecord : inherits
    class IWRecord {
        <<interface>>
        ~Kind => TypeKind_WRecord
    }

    IRecord <|-- IORecord : inherits
    class IORecord {
        <<interface>>
        ~Kind => TypeKind_ORecord
    }

    IObject --|> IStructure : inherits
    class IObject {
        <<interface>>
        ~Kind => TypeKind_Object
    }

    IElement --|> IStructure : inherits
    class IElement {
        <<interface>>
        ~Kind => TypeKind_Element
    }

    IType <|-- IView : inherits
    class IView {
        <<interface>>
        ~Kind => TypeKind_ViewRecord
        +Key() IViewKey
        +Value() IViewValue
    }
            
    IType <|-- IFunc : inherits
    class IFunc {
        <<interface>>
        +Params() []IParam
        +Results() []IParam
    }

    IFunc <|-- ICommand : inherits
    class ICommand {
        <<interface>>
        ~Kind => TypeKind_Command
        +UnloggedParams() []IParam
    }

    IFunc <|-- IQuery : inherits
    class IQuery {
        <<interface>>
        ~Kind => TypeKind_Query
    }


    IWorkspace --|> IType : inherits
    class IWorkspace {
        <<interface>>
        ~Kind => TypeKind_Workspace
        +Abstract() bool
        +Types() []IType
    }
```

### Types composing

```mermaid
classDiagram


    class IData {
        <<interface>>
        ~Kind => TypeKind_Data
        +Ancestor() IData
        +Restricts() []IDataRestrict
    }
    IData "1" ..> "0..1" IData : ancestor ref

    class IArray {
        <<interface>>
        ~Kind => TypeKind_Array
        +MaxLen() uint
        +Elem() IType
    }

    class IGDoc {
        <<interface>>
        ~Kind => TypeKind_GDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IGDoc "1" o..> "0..*" IGRecord : contains

    class ICDoc {
        <<interface>>
        ~Kind => TypeKind_CDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
        +Singleton() bool
    }
    ICDoc "1" o..> "0..*" ICRecord : contains

    class IWDoc {
        <<interface>>
        ~Kind => TypeKind_WDoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IWDoc "1" o..> "0..*" IWRecord : contains
                    
    class IODoc {
        <<interface>>
        ~Kind => TypeKind_ODoc
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IODoc "1" o..> "0..*" IODoc : contains
    IODoc "1" o..> "0..*" IORecord : contains

    class IGRecord {
        <<interface>>
        ~Kind => TypeKind_GRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IGRecord "1" o..> "0..*" IGRecord : contains

    class ICRecord {
        <<interface>>
        ~Kind => TypeKind_CRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    ICRecord "1" o..> "0..*" ICRecord : contains

    class IWRecord {
        <<interface>>
        ~Kind => TypeKind_WRecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
        +Uniques() []IUnique
    }
    IWRecord "1" o..> "0..*" IWRecord : contains

    class IORecord {
        <<interface>>
        ~Kind => TypeKind_ORecord
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IORecord "1" o..> "0..*" IORecord : contains

    class IObject {
        <<interface>>
        ~Kind => TypeKind_Object
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IObject "1" o..> "0..*" IElement : contains

    class IElement {
        <<interface>>
        ~Kind => TypeKind_Element
        +Abstract() bool
        +Fields() []IField
        +Containers() []IContainer
    }
    IElement "1" o..> "0..*" IElement : contains

    class IView {
        <<interface>>
        ~Kind => TypeKind_ViewRecord
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
        ~Kind => TypeKind_Command
        +Params() []IParam
        +UnloggedParams() []IParam
        +Results() []IParam
    }

    class IQuery {
        <<interface>>
        ~Kind => TypeKind_Query
        +Params() []IParam
        +Results() []IParam
    }

    class IWorkspace {
        <<interface>>
        ~Kind => TypeKind_Workspace
        +Abstract() bool
        +Types() []IType
    }

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

### Structures

```mermaid
classDiagram
  direction BT

  class IDef{
    <<Interface>>
    +Kind() DefKind
    +QName() QName
  }

  IGDoc --|> IDef : inherits
  class IGDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IGDoc "1" --o "0..*" IGRecord : aggregate child

  IGDocBuilder --|> IGDoc : inherits
  class IGDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IGRecord --|> IDef : inherits
  class IGRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IGRecord "1" --o "0..*" IGRecord : aggregate child

  IGRecordBuilder --|> IGRecord : inherits
  class IGRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  ICDoc --|> IDef : inherits
  class ICDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
    +Singleton() bool
  }
  ICDoc "1" --o "0..*" ICRecord : aggregate child

  ICDocBuilder --|> ICDoc : inherits
  class ICDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
    +SetSingleton()
  }

  ICRecord --|> IDef : inherits
  class ICRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  ICRecord "1" --o "0..*" ICRecord : aggregate child

  ICRecordBuilder --|> ICRecord : inherits
  class ICRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IWDoc --|> IDef : inherits
  class IWDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IWDoc "1" --o "0..*" IWRecord : aggregate child

  IWDocBuilder --|> IWDoc : inherits
  class IWDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IWRecord --|> IDef : inherits
  class IWRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IWRecord "1" --o "0..*" IWRecord : aggregate child

  IWRecordBuilder --|> IWRecord : inherits
  class IWRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IODoc --|> IDef : inherits
  class IODoc {
    <<Interface>>
    IFields
    IContainers
  }
  IODoc "1" --o "0..*" IODoc : aggregate child docs
  IODoc "1" --o "0..*" IORecord : aggregate child recs

  IODocBuilder --|> IODoc : inherits
  class IODocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }

  IORecord --|> IDef : inherits
  class IORecord {
    <<Interface>>
    IFields
    IContainers
  }
  IORecord "1" --o "0..*" IORecord : aggregate child

  IORecordBuilder --|> IORecord : inherits
  class IORecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }

  IObject --|> IDef : inherits

  class IObject {
    <<Interface>>
    IFields
    IContainers
  }
  IObject "1" --o "0..*" IElement : aggregate child

  IObjectBuilder --|> IObject : inherits
  class IObjectBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }
  
  IElement --|> IDef : inherits
  class IElement {
    <<Interface>>
    IFields
    IContainers
  }
  IElement "1" --o "0..*" IElement : aggregate child

  IElementBuilder --|> IElement : inherits
  class IElementBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }
```

### Views

```mermaid
classDiagram
  direction BT

  class IDef{
    <<Interface>>
    +Kind() DefKind
    +QName() QName
  }

  class IField {
    <<Interface>>
    +Name() string
    +DataKind() DataKind
    +Required() bool
  }
  note for IField "Required is always true \n for partition key fields, \n false for clustering columns \n and optional for value fields"

  IView --|> IDef : inherits
  class IView {
    <<Interface>>
    IContainers
    +PartKey() IPartKey
    +ClustCols() IClustCols
    +Key() IViewKey
    +Value() IViewValue
  }
  IView --o IPartKey : aggregate
  IView --o IClustCols : aggregate
  IView --o IViewKey : aggregate
  IView --o IViewValue : aggregate

  IPartKey --|> IDef : inherits
  class IPartKey {
    <<Interface>>
    IFields
  }
  IPartKey "1" -- "1..*" IField : compose

  IClustCols --|> IDef : inherits
  class IClustCols {
    <<Interface>>
    IFields
  }
  IClustCols "1" -- "1..*" IField : compose

  IViewKey --|> IDef : inherits
  class IViewKey {
    <<Interface>>
    IFields
  }
  IViewKey "1" -- "1..*" IField : compose

  IViewValue --|> IDef : inherits
  class IViewValue {
    <<Interface>>
    IFields
  }
  IViewValue "1" -- "1..*" IField : compose
```

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

- Maximum fields per definition is 65536.
- Maximum string field length is 1024.

### Containers

- Maximum containers per definition is 65536.

### Uniques

- Maximum fields per unique is 256
- Maximum uniques per definition is 100.
