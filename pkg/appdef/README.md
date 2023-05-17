# Application Definition

[![codecov](https://codecov.io/gh/voedger/voedger/appdef/branch/main/graph/badge.svg?token=u6VrbqKtnn)](https://codecov.io/gh/voedger/voedger/appdef)

## Definitions inheritance

### Overview

```mermaid
mindmap
  root((Def))
    Structures["Structures"]
      GDoc("GDoc")
        (GRec)
      CDoc("CDoc")
        (CRec)
      (WDoc)
        (WRec)
      (ODoc)
        (ORec)
      (Object)
        (Element)
    [Views]
      (PartKey)
      (ClustCols)
      (Value)
    [Resources]
      (Commands)
      (Queries)
```

### Fields, Containers, Uniques

```mermaid
classDiagram

direction TB

  class IFields{
    <<Interface>>
    Field(string) IField
    FieldCount() int
    Fields(func(IField))
  }
  IFields "1" --* "0..*" IField : compose

  IFieldsBuilder <|-- IFields : inherits
  class IFieldsBuilder {
    <<Interface>>
    AddField(…) IField
    AddVerifiedField(…) IField
  }

  class IField {
    <<Interface>>
    +Name() string
    +DataKind() DataKind
    +Required() bool
    +Verified() bool
    +VerificationKind(VerificationKind) bool
  }

  class IContainers{
    <<Interface>>
    Container(string) IContainer
    ContainerCount() int
    ContainerDef(string) IDef
    Containers(func(IContainer))
  }
  IContainers "1" --* "0..*" IContainer : compose

  IContainersBuilder <|-- IContainers : inherits
  class IContainersBuilder {
    <<Interface>>
    AddContainer(…) IContainer
  }

  class IContainer {
    <<Interface>>
    +Name() string
    +Def() IDef
    +MinOccurs() int
    +MaxOccurs() int
  }

  class IUniques{
    <<Interface>>
    UniqueByID(UniqueID) IUnique
    UniqueByName(string) IUnique
    UniqueCount() int
    Uniques(func(IUnique))
  }
  IUniques "1" --* "0..*" IUnique : compose

  IUniquesBuilder <|-- IUniques : inherits
  class IUniquesBuilder {
    <<Interface>>
    AddUnique(…) IUnique
  }

  class IUnique {
    <<Interface>>
    +Name() string
    +ID() UniqueID
    +Fields() []IFeld
  }
```

### Structures

```mermaid
classDiagram

direction TB
  class IDef{
    <<Interface>>
    +Kind() DefKind
    +QName() QName
  }

  IGDoc <|-- IDef : inherits
  class IGDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IGDoc "1" --o "0..*" IGRecord : aggregate child

  IGDocBuilder <|-- IGDoc : inherits
  class IGDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IGRecord <|-- IDef : inherits
  class IGRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IGRecord "1" --o "0..*" IGRecord : aggregate child

  IGRecordBuilder <|-- IGRecord : inherits
  class IGRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  ICDoc <|-- IDef : inherits
  class ICDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
    +Singleton() bool
  }
  ICDoc "1" --o "0..*" ICRecord : aggregate child

  ICDocBuilder <|-- ICDoc : inherits
  class ICDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
    +SetSingleton()
  }

  ICRecord <|-- IDef : inherits
  class ICRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  ICRecord "1" --o "0..*" ICRecord : aggregate child

  ICRecordBuilder <|-- ICRecord : inherits
  class ICRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IWDoc <|-- IDef : inherits
  class IWDoc {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IWDoc "1" --o "0..*" IWRecord : aggregate child

  IWDocBuilder <|-- IWDoc : inherits
  class IWDocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IWRecord <|-- IDef : inherits
  class IWRecord {
    <<Interface>>
    IFields
    IContainers
    IUniques
  }
  IWRecord "1" --o "0..*" IWRecord : aggregate child

  IWRecordBuilder <|-- IWRecord : inherits
  class IWRecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
    IUniquesBuilder
  }

  IODoc <|-- IDef : inherits
  class IODoc {
    <<Interface>>
    IFields
    IContainers
  }
  IODoc "1" --o "0..*" IODoc : aggregate child docs
  IODoc "1" --o "0..*" IORecord : aggregate child recs

  IODocBuilder <|-- IODoc : inherits
  class IODocBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }

  IORecord <|-- IDef : inherits
  class IORecord {
    <<Interface>>
    IFields
    IContainers
  }
  IORecord "1" --o "0..*" IORecord : aggregate child

  IORecordBuilder <|-- IORecord : inherits
  class IORecordBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }

  IObject <|-- IDef : inherits

  class IObject {
    <<Interface>>
    IFields
    IContainers
  }
  IObject "1" --o "0..*" IElement : aggregate child

  IObjectBuilder <|-- IObject : inherits
  class IObjectBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }
  
  IElement <|-- IDef : inherits
  class IElement {
    <<Interface>>
    IFields
    IContainers
  }
  IElement "1" --o "0..*" IElement : aggregate child

  IElementBuilder <|-- IElement : inherits
  class IElementBuilder {
    <<Interface>>
    IFieldsBuilder
    IContainersBuilder
  }
```

### Views

```mermaid
classDiagram
direction TB
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

  IView <|-- IDef : inherits
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

  IPartKey <|-- IDef : inherits
  class IPartKey {
    <<Interface>>
    IFields
  }
  IPartKey "1" -- "1..*" IField : compose

  IClustCols <|-- IDef : inherits
  class IClustCols {
    <<Interface>>
    IFields
  }
  IClustCols "1" -- "1..*" IField : compose

  IViewKey <|-- IDef : inherits
  class IViewKey {
    <<Interface>>
    IFields
  }
  IViewKey "1" -- "1..*" IField : compose

  IViewValue <|-- IDef : inherits
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

### Containers

- Maximum containers per definition is 65536.

### Uniques

- Maximum fields per unique is 256
- Maximum uniques per definition is 100.
