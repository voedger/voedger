# Compatibility 

Motivation
- [Parser: Package AST compatibility tool](https://github.com/voedger/voedger/issues/617)


## Functional Design

### Concepts

Compatibility error types:

- Read compatibility (some reads will fail)
  - Table is removed from a workspace
  - Table is removed from a package
  - Table usage is removed form a workspace
  - Field order is changed in a Table
  - Field is added to a View key
  - Table is removed from a workspace scope
  - Field type is changed
- API compatibility  (some writes will fail)
  - Constraint is added/changed/removed
  - ACL entry is added or changed

### Principles

- Only  Storage compatibility errors are checked

  
### Functions

```go
// err: errors.Join of all compatibility errors
func CheckPackageBackwardCompatibility(old, new *PackageSchemaAST) (cerrs CompatibilityErrors)

// err: errors.Join of all compatibility errors
func CheckApplicationBackwardCompatibility(old, new *PackageSchemaAST) (cerrs CompatibilityErrors)


// cerrsOut: all cerrsIn that are not in toBeIgnored
// toBeIgnored.Pos is ignored in comparison
func IgnoreCompatibilityErrors(cerrs CompatibilityErrors, toBeIgnored []CompatibilityError) (cerrsOut CompatibilityErrors)
```

Errors are kept in parser.errors_compat.go

## Technical Design

### Algorythm
1. Build old and new `CompatibilityTree`-s
2. Compare CompatibilityTree-s
  1. CompareTrees(oldTree, newTree) (cerrsOut CompatibilityErrors)


### Build CompatibilityTree

```go
func buildPackageCompatibilityNode(treeCtx TreeCtx, package *PackageSchemaAST) {
  treeCtx := treeCtx.Add("sql.PACKAGE", package.Name)
  for _, stmt := range package.Statements {
    switch stmt := stmt.(type) {
    case *sql.Table:
      buildTableCompatibilityNode(treeCtx, stmt)
    case *sql.View:
      buildViewCompatibilityNode(treeCtx, stmt)
    case *sql.Workspace:
      // Create or add to existing workspace node
      treeCtxWS := treeCtx.AddIfNotExist("sql.WORKSPACE", stmt.Name)
      buildWorkspaceCompatibilityNode(treeCtx, stmt)
    case *sql.AlterWorkspaceStmt:
      // Create or add to existing workspace node
      treeCtxWS := treeCtx.AddIfNotExist("sql.WORKSPACE", stmt.Name)
      buildWorkspaceCompatibilityNode(treeCtx, stmt)      
    case *sql.Package:
      buildPackageCompatibilityNode(treeCtx, stmt)
    }
  } 
}  
```

### Types

```go
type CompatibilityTreeNode {
    
    Name string
    Type NodeType // "Prop", "Props", "sql.TABLE", "sql.VIEW", "sql.WORKSPACE", "sql.PACKAGE"

    // For "Props" and "sql.*" nodes
    // Key is Name + Type
    Props []CompatibilityTreeNode

    // For "Prop" nodes
    Value interface{}
}

type NodeConstraint struct {
    NodeType NodeType
    PropName string
    
    // "ConstraintAppendOnly" means that Props elements can only be appended, order cannot be changed
    // "ConstraintInsertOnly" means that Props elements can only be inserted, order CAN be changed
    // "ConstraintNonModifiable" means that Props elements may not be added or removed, value of Prop may not be changed
    Constraint Constraint 
}

```
Package CompatibilityTree example

- Package Package
  - Statements Props
    - MyTable1 sql.TABLE
      - Ancestor Prop
      - Items Props
      
    - MyView1  sql.VIEW
      - PKFields Props
        - PKField1 sql.FIELD
          - Type   Prop
        - PKField2 sql.FIELD
      - CCFields Props
        - CCField1 FIELD
        - CCField1 FIELD 
      - ValueFields []Field
        - ValueField1 Field
        - ValueField2 Field
    - MyTable2 Table
    - MyView2  View
    - MyWorkspace WORKSPACE
      - Statements
        - 
NodeConstraint examples:
```golang

// Statements can be added to package
packageStatementsConstraint := NodeConstraint{"sql.PACKAGE", "Statements", ConstraintInsertOnly} 

// Statements can be added to workspace
workspaceStatementsConstraint := NodeConstraint{"sql.WORKSPACE", "Statements", ConstraintInsertOnly}

// Fields can be appended, non added
tableItemsConstraint := NodeConstraint{"sql.TABLE", "Items", ConstraintAppendOnly}

// Table ancestor may not be changed. This is the default constraint, so it can be ommited
tableAncestorConstraint := NodeConstraint{"sql.TABLE", "Ancestor", ConstraintNonModifiable}

// Table parent may not be changed. This is the default constraint, so it can be ommited
tableParentConstraint := NodeConstraint{"sql.TABLE", "Parent", ConstraintNonModifiable}


viewValueFieldsConstraint := NodeConstraint{"View", "ValueFields", ConstraintAppend}

type CompatibilityError struct {
    Constraint Constraint

    // Position in old AST
    Pos lexer.Position

    // Position in old AST
    Path []PathElement

    // Name of the previous element when order is changed, "" if element becomes the first
    // Name of the element that is removed
    ErrorValue string
}

type PathElement struct {
    Type string
    Name string
}

type CompatibilityErrors {
    Errors []CompatibilityError
    Error() string
}
```
