# Compatibility 

Motivation
- [Parser: Package AST compatibility tool](https://github.com/voedger/voedger/issues/617)

## Functional Design

### Concepts

Compatibility error types:

- Projector Read compatibility (some projectors reads fail or even crash). Some examples:
  - Table is removed from a workspace
  - Table is removed from a package
  - Table usage is removed form a workspace
  - Field order is changed in a Table
  - Field is added to a View key
  - Table is removed from a workspace scope
  - Field type is changed
- Projector Write compatibility  (some projectors writes fail). Some examples:
  - Constraint is added/changed/removed
- Other possible compatibility errors examples:
  - ACL entry is added or changed, it changes API behaviour for non-System authorization

Why "Projector..."? To simplify definition, Projector uses System authorization and is not affected by ACL changes.



### Principles

- Only Projector Read compatibility errors are checked
  
### Functions

```go

func CheckBackwardCompatibility(oldAppDef, newAppDef appdef.IAppDef) (cerrs *CompatibilityErrors)

func IgnoreCompatibilityErrors(cerrs *CompatibilityErrors, pathsToIgnore [][]string) (cerrsOut *CompatibilityErrors)
```

## Technical Design

### Principles

- appdef.AppDef is used, not parser.ASTs
  - Rationale: appdef.AppDef is easier to use, e.g. 
    - Parser.PackageAST contains multiple definitions of Workspace that must be merged before use
    - AST does not provide list of all QNames in a package
- Algorythm
  1. Build old and new `CompatibilityTree`-s
  2. Compare CompatibilityTree-s using `NodeConstraint`-s
    

### Tree and Constraints

```go
type Constraint string
type NodeType string

const (
	ConstraintValueMatch    Constraint = "ConstraintValueMatch"
    ConstraintAppendOnly    Constraint = "ConstraintAppendOnly"
    ConstraintInsertOnly    Constraint = "ConstraintInsertOnly"
    ConstraintNonModifiable Constraint = "ConstraintNonModifiable"
)

type CompatibilityTreeNode {
    ParentNode *CompatibilityTreeNode
    Name string
    Props []*CompatibilityTreeNode
    Value interface{}
    invisibleInPath bool
}

type NodeConstraint struct {
    NodeName string
    Constraint Constraint
}
```

### CompatibilityTree example

- AppDef AppDef
  - Packages
    - packagePath1 LocalName1
    - packagePath2 LocalName2
    - packagePath3 LocalName3
    - packagePath4 LocalName4
    - packagePath5 LocalName5
  - Types
    - pkg1.Workspace1 // IWorkspace
      - Types
        - pkg1.Table5
        - pkg5.View2
      - Inheritance // FIXME not implemented???
        - pkg1.Workspace2
        - pkg1.Workspace3
      - Descriptor pkg1.Workspace1Descriptor
    - pkg2.SomeQName // IType
      - Abstract true // IWithAbstract
      - Fields // IFields
        - Name1 int // IField
        - Name2 varchar (no length here) // IField
      - Containers // IContainers
        - Name1 QName1
        - Name2 QName2
    - pkg3.SomeTable // IDoc
      - Abstract true // IWithAbstract
      - Fields // IFields
        - Name1 int // IField
        - Name2 varchar (no length here) // IField
      - Containers // IContainers
        - Name1 QName1
        - Name2 QName2
      - Uniques // IUniques
        - Name1 QName1 // IUnique
          - UniqueFields
            - Name2 varchar
            - Name1 int
          - Parent
            - Abstract true // IWithAbstract
            - Fields // IFields
              - Name1 int
              - Name2 varchar (no length here)
            - Containers // IContainers
              - Name1 QName1
              - Name2 QName2
        - Name2 QName2 // IUnique
          -  UniqueFields
            - Name1 int
            - Name2 varchar
          - Parent
            - Abstract true // IWithAbstract
            - Fields // IFields
              - Name1 int
              - Name2 varchar (no length here)
            - Containers // IContainers
              - Name1 QName1
              - Name2 QName2
    - pkg3.View // IView
      - PartKeyFields // Key().Partition()
         - Name1 int
         - Name2 int
      - ClustColsFields // Key().ClustCols()
        - Name1 int
        - Name2 varchar
      - Fields // Value fields 
        - ...
      // FIXME Containers ???
    - pkg3.Projector Props
      - Sync true
    -pkg3.Command Props // ICommand
      - CommandArgs Props      
      - UnloggedArgs Props
      - CommandResult Props
    - pkg3.Query // IQuery
      - QueryArgs Props      
      - QueryResult Props

### Constraints

NodeConstraint examples:
```golang

typesConstraint := NodeConstraint{"Types", ConstraintInsertOnly}
fieldsConstraint := NodeConstraint{"Fields", ConstraintAppendOnly}
```

### CompatibilityError

```golang
type CompatibilityError struct {
    Constraint Constraint

    OldTreePath []string

    // NodeRemoved:  (NonModifiable, AppendOnly,InsertOnly) : one error per removed node
    // OrderChanged: (NonModifiable, AppendOnly): one error for the container
    // NodeInserted: (NonModifiable): one error for the container
	// ValueChanged: one error for one node
	// NodeModified: one error for the container
    ErrorType ErrorType
}
```
OldTreePath example:
```golang
[]string{"AppDef", "Types", "sys.Workspace1", "Types", "sys.Table5", "Fields", "Name1"}
```

```golang
type CompatibilityErrors {
    Errors []CompatibilityError
    Error() string
}
```