/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	fs "io/fs"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/voedger/voedger/pkg/appdef"
)

type FileSchemaAST struct {
	FileName string
	Ast      *SchemaAST
}

type PackageSchemaAST struct {
	QualifiedPackageName string
	Ast                  *SchemaAST
}

type IReadFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type Ident string

func (b *Ident) Capture(values []string) error {
	*b = Ident(strings.Trim(values[0], "\""))
	return nil
}

type IStatement interface {
	GetPos() *lexer.Position
	GetComments() *[]string
}

type INamedStatement interface {
	IStatement
	GetName() string
}
type IStatementCollection interface {
	Iterate(callback func(stmt interface{}))
}
type IExtensionStatement interface {
	SetEngineType(EngineType)
}

type SchemaAST struct {
	Package    Ident           `parser:"'SCHEMA' @Ident ';'"`
	Imports    []ImportStmt    `parser:"@@? (';' @@)* ';'?"`
	Statements []RootStatement `parser:"@@? (';' @@)* ';'?"`
}

func (s *SchemaAST) NewQName(name Ident) appdef.QName {
	return appdef.NewQName(string(s.Package), string(name))
}

func (s *SchemaAST) Iterate(callback func(stmt interface{})) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = extractStatement(*raw)
		}
		callback(raw.stmt)
	}
}

type ImportStmt struct {
	Pos   lexer.Position
	Name  string `parser:"'IMPORT' 'SCHEMA' @String"`
	Alias *Ident `parser:"('AS' @Ident)?"`
}

type RootStatement struct {
	// Only allowed in root
	Template *TemplateStmt `parser:"@@"`

	// Also allowed in root
	Role      *RoleStmt          `parser:"| @@"`
	Tag       *TagStmt           `parser:"| @@"`
	ExtEngine *RootExtEngineStmt `parser:"| @@"`
	Workspace *WorkspaceStmt     `parser:"| @@"`
	Table     *TableStmt         `parser:"| @@"`
	Type      *TypeStmt          `parser:"| @@"`
	// Sequence  *sequenceStmt  `parser:"| @@"`

	stmt interface{}
}

type WorkspaceStatement struct {
	// Only allowed in workspace
	Rate     *RateStmt     `parser:"@@"`
	View     *ViewStmt     `parser:"| @@"`
	UseTable *UseTableStmt `parser:"| @@"`

	// Also allowed in workspace
	Role      *RoleStmt               `parser:"| @@"`
	Tag       *TagStmt                `parser:"| @@"`
	ExtEngine *WorkspaceExtEngineStmt `parser:"| @@"`
	Workspace *WorkspaceStmt          `parser:"| @@"`
	Table     *TableStmt              `parser:"| @@"`
	Type      *TypeStmt               `parser:"| @@"`
	//Sequence  *sequenceStmt  `parser:"| @@"`
	Grant *GrantStmt `parser:"| @@"`

	stmt interface{}
}

type RootExtEngineStatement struct {
	Function *FunctionStmt `parser:"@@"`
	Storage  *StorageStmt  `parser:"| @@"`
	stmt     interface{}
}

type WorkspaceExtEngineStatement struct {
	Function  *FunctionStmt  `parser:"@@"`
	Projector *ProjectorStmt `parser:"| @@"`
	Command   *CommandStmt   `parser:"| @@"`
	Query     *QueryStmt     `parser:"| @@"`
	stmt      interface{}
}

type WorkspaceExtEngineStmt struct {
	Engine     EngineType                    `parser:"EXTENSIONENGINE @@"`
	Statements []WorkspaceExtEngineStatement `parser:"'(' @@? (';' @@)* ';'? ')'"`
}

func (s *WorkspaceExtEngineStmt) Iterate(callback func(stmt interface{})) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = extractStatement(*raw)
			if es, ok := raw.stmt.(IExtensionStatement); ok {
				es.SetEngineType(s.Engine)
			}
		}
		callback(raw.stmt)
	}
}

type RootExtEngineStmt struct {
	Engine     EngineType               `parser:"EXTENSIONENGINE @@"`
	Statements []RootExtEngineStatement `parser:"'(' @@? (';' @@)* ';'? ')'"`
}

func (s *RootExtEngineStmt) Iterate(callback func(stmt interface{})) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = extractStatement(*raw)
			if es, ok := raw.stmt.(IExtensionStatement); ok {
				es.SetEngineType(s.Engine)
			}
		}
		callback(raw.stmt)
	}
}

type WorkspaceStmt struct {
	Statement
	Abstract   bool                 `parser:"@'ABSTRACT'?"`
	Pool       bool                 `parser:"@('POOL' 'OF')?"`
	Name       Ident                `parser:"'WORKSPACE' @Ident "`
	Inherits   *DefQName            `parser:"('INHERITS' @@)?"`
	A          int                  `parser:"'('"`
	Descriptor *WsDescriptorStmt    `parser:"('DESCRIPTOR' @@)?"`
	Statements []WorkspaceStatement `parser:"@@? (';' @@)* ';'? ')'"`
}

func (s WorkspaceStmt) GetName() string { return string(s.Name) }
func (s *WorkspaceStmt) Iterate(callback func(stmt interface{})) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = extractStatement(*raw)
		}
		callback(raw.stmt)
	}
}

type TypeStmt struct {
	Statement
	Name  Ident           `parser:"'TYPE' @Ident "`
	Items []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
}

func (s TypeStmt) GetName() string { return string(s.Name) }

type WsDescriptorStmt struct {
	Statement
	Items []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
	_     int             `parser:"';'"`
}

type DefQName struct {
	Package Ident `parser:"(@Ident '.')?"`
	Name    Ident `parser:"@Ident"`
}

func (q DefQName) String() string {
	if q.Package == "" {
		return string(q.Name)
	}
	return fmt.Sprintf("%s.%s", q.Package, q.Name)

}

type TypeQName struct {
	Package Ident `parser:"(@Ident '.')?"`
	Name    Ident `parser:"@Ident"`
	IsArray bool  `parser:"@Array?"`
}

func (q TypeQName) String() (s string) {
	if q.Package == "" {
		s = string(q.Name)
	} else {
		s = fmt.Sprintf("%s.%s", q.Package, q.Name)
	}

	if q.IsArray {
		return fmt.Sprintf("[]%s", s)
	}
	return s
}

type Statement struct {
	Pos      lexer.Position
	Comments []string `parser:"@PreStmtComment*"`
}

func (s *Statement) GetPos() *lexer.Position {
	return &s.Pos
}

func (s *Statement) GetComments() *[]string {
	return &s.Comments
}

type StorageKey struct {
	Storage DefQName  `parser:"@@"`
	Entity  *DefQName `parser:"( @@ )?"`
}

type ProjectorStmt struct {
	Statement
	Sync     bool         `parser:"@'SYNC'?"`
	Name     Ident        `parser:"'PROJECTOR' @Ident"`
	On       ProjectorOn  `parser:"'ON' @@"`
	Triggers []DefQName   `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	State    []StorageKey `parser:"('STATE'   '(' @@ (',' @@)* ')' )?"`
	Intents  []StorageKey `parser:"('INTENTS' '(' @@ (',' @@)* ')' )?"`
	Engine   EngineType   // Initialized with 1st pass
}

func (s *ProjectorStmt) GetName() string            { return string(s.Name) }
func (s *ProjectorStmt) SetEngineType(e EngineType) { s.Engine = e }

type ProjectorOn struct {
	CommandArgument bool `parser:"@('COMMAND' 'ARGUMENT')"`
	Command         bool `parser:"| @('COMMAND')"`
	Insert          bool `parser:"| @(('INSERT' ('OR' 'UPDATE')?) | ('UPDATE' 'OR' 'INSERT'))"`
	Update          bool `parser:"| @(('UPDATE' ('OR' 'INSERT')?) | ('INSERT' 'OR' 'UPDATE'))"`
	Activate        bool `parser:"| @(('ACTIVATE' ('OR' 'DEACTIVATE')?) | ('DEACTIVATE' 'OR' 'ACTIVATE'))"`
	Deactivate      bool `parser:"| @(('DEACTIVATE' ('OR' 'ACTIVATE')?) | ('ACTIVATE' 'OR' 'DEACTIVATE'))"`
}

type TemplateStmt struct {
	Statement
	Name      Ident    `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace DefQName `parser:"@@"`
	Source    Ident    `parser:"'SOURCE' @Ident"`
}

func (s TemplateStmt) GetName() string { return string(s.Name) }

type RoleStmt struct {
	Statement
	Name Ident `parser:"'ROLE' @Ident"`
}

func (s RoleStmt) GetName() string { return string(s.Name) }

type TagStmt struct {
	Statement
	Name Ident `parser:"'TAG' @Ident"`
}

func (s TagStmt) GetName() string { return string(s.Name) }

type UseTableStmt struct {
	Statement
	Package   Ident `parser:"'USE' 'TABLE' (@Ident '.')?"`
	Name      Ident `parser:"(@Ident "`
	AllTables bool  `parser:"| @'*')"`
}

type UseTableItem struct {
	Package   Ident `parser:"(@Ident '.')?"`
	Name      Ident `parser:"(@Ident "`
	AllTables bool  `parser:"| @'*')"`
}

/*type sequenceStmt struct {
	Name        Ident `parser:"'SEQUENCE' @Ident"`
	Type        Ident `parser:"@Ident"`
	StartWith   *int   `parser:"(('START' 'WITH' @Number)"`
	MinValue    *int   `parser:"| ('MINVALUE' @Number)"`
	MaxValue    *int   `parser:"| ('MAXVALUE' @Number)"`
	IncrementBy *int   `parser:"| ('INCREMENT' 'BY' @Number) )*"`
}*/

type RateStmt struct {
	Statement
	Name   Ident  `parser:"'RATE' @Ident"`
	Amount int    `parser:"@Int"`
	Per    string `parser:"'PER' @('SECOND' | 'MINUTE' | 'HOUR' | 'DAY' | 'YEAR')"`
	PerIP  bool   `parser:"(@('PER' 'IP'))?"`
}

func (s RateStmt) GetName() string { return string(s.Name) }

type GrantStmt struct {
	Statement
	Grants []string `parser:"'GRANT' @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE') (','  @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE'))*"`
	On     string   `parser:"'ON' @('TABLE' | ('ALL' 'TABLES' 'WITH' 'TAG') | 'COMMAND' | ('ALL' 'COMMANDS' 'WITH' 'TAG') | 'QUERY' | ('ALL' 'QUERIES' 'WITH' 'TAG'))"`
	Target DefQName `parser:"@@"`
	To     Ident    `parser:"'TO' @Ident"`
}

type StorageStmt struct {
	Statement
	Name         Ident       `parser:"'STORAGE' @Ident"`
	Ops          []StorageOp `parser:"'(' @@ (',' @@)* ')'"`
	EntityRecord bool        `parser:"@('ENTITY' 'RECORD')?"`
	EntityView   bool        `parser:"@('ENTITY' 'VIEW')?"`
}

func (s StorageStmt) GetName() string { return string(s.Name) }

type StorageOp struct {
	Get      bool           `parser:"( @'GET'"`
	GetBatch bool           `parser:"| @'GETBATCH'"`
	Read     bool           `parser:"| @'READ'"`
	Insert   bool           `parser:"| @'INSERT'"`
	Update   bool           `parser:"| @'UPDATE')"`
	Scope    []StorageScope `parser:"'SCOPE' '(' @@ (',' @@)* ')'"`
}

type StorageScope struct {
	Commands   bool `parser:" ( @'COMMANDS'"`
	Queries    bool `parser:" | @'QUERIES'"`
	Projectors bool `parser:" | @'PROJECTORS')"`
}

type FunctionStmt struct {
	Statement
	Name    Ident           `parser:"'FUNCTION' @Ident"`
	Params  []FunctionParam `parser:"'(' @@? (',' @@)* ')'"`
	Returns TypeQName       `parser:"'RETURNS' @@"`
	Engine  EngineType      // Initialized with 1st pass
}

func (s *FunctionStmt) GetName() string            { return string(s.Name) }
func (s *FunctionStmt) SetEngineType(e EngineType) { s.Engine = e }

type CommandStmt struct {
	Statement
	Name        Ident      `parser:"'COMMAND' @Ident"`
	Arg         *DefQName  `parser:"('(' @@? "`
	UnloggedArg *DefQName  `parser:"(','? UNLOGGED @@)? ')')?"`
	Returns     *DefQName  `parser:"('RETURNS' @@)?"`
	With        []WithItem `parser:"('WITH' @@ (',' @@)* )?"`
	Engine      EngineType // Initialized with 1st pass
}

func (s *CommandStmt) GetName() string            { return string(s.Name) }
func (s *CommandStmt) SetEngineType(e EngineType) { s.Engine = e }

type WithItem struct {
	Comment *string    `parser:"('Comment' '=' @String)"`
	Tags    []DefQName `parser:"| ('Tags' '=' '(' @@ (',' @@)* ')')"`
	Rate    *DefQName  `parser:"| ('Rate' '=' @@)"`
}

type QueryStmt struct {
	Statement
	Name    Ident      `parser:"'QUERY' @Ident"`
	Arg     *DefQName  `parser:"('(' @@? ')')?"`
	Returns DefQName   `parser:"'RETURNS' @@"`
	With    []WithItem `parser:"('WITH' @@ (',' @@)* )?"`
	Engine  EngineType // Initialized with 1st pass
}

func (s *QueryStmt) GetName() string            { return string(s.Name) }
func (s *QueryStmt) SetEngineType(e EngineType) { s.Engine = e }

type EngineType struct {
	WASM    bool `parser:"@'WASM'"`
	Builtin bool `parser:"| @'BUILTIN'"`
}

type FunctionParam struct {
	NamedParam       *NamedParam `parser:"@@"`
	UnnamedParamType *TypeQName  `parser:"| @@"`
}

type NamedParam struct {
	Name Ident     `parser:"@Ident"`
	Type TypeQName `parser:"@@"`
}

type TableStmt struct {
	Statement
	Name         Ident           `parser:"'TABLE' @Ident"`
	Inherits     *DefQName       `parser:"('INHERITS' @@)?"`
	Items        []TableItemExpr `parser:"'(' @@? (',' @@)* ')'"`
	With         []WithItem      `parser:"('WITH' @@ (',' @@)* )?"`
	tableDefKind appdef.DefKind  // filled on the analysis stage
	singletone   bool
}

func (s TableStmt) GetName() string { return string(s.Name) }

type NestedTableStmt struct {
	Pos   lexer.Position
	Name  Ident     `parser:"@Ident"`
	Table TableStmt `parser:"@@"`
}

type FieldSetItem struct {
	Pos  lexer.Position
	Type DefQName `parser:"@@"`
}

type TableItemExpr struct {
	NestedTable *NestedTableStmt `parser:"@@"`
	Constraint  *TableConstraint `parser:"| @@"`
	RefField    *RefFieldExpr    `parser:"| @@"`
	Field       *FieldExpr       `parser:"| @@"`
	FieldSet    *FieldSetItem    `parser:"| @@"`
}

type TableConstraint struct {
	Pos            lexer.Position
	ConstraintName Ident            `parser:"('CONSTRAINT' @Ident)?"`
	UniqueField    *UniqueFieldExpr `parser:"(@@"`
	//	Unique         *UniqueExpr      `parser:"(@@"` // TODO: not supported by kernel yet
	Check *TableCheckExpr `parser:"| @@)"`
}

type TableCheckExpr struct {
	Expression Expression `parser:"'CHECK' '(' @@ ')'"`
}

type UniqueFieldExpr struct {
	Field Ident `parser:"'UNIQUEFIELD' @Ident"`
}

type UniqueExpr struct {
	Fields []Ident `parser:"'UNIQUE' '(' @Ident (',' @Ident)* ')'"`
}

type RefFieldExpr struct {
	Pos     lexer.Position
	Name    Ident      `parser:"@Ident"`
	RefDocs []DefQName `parser:"'ref' ('(' @@ (',' @@)* ')')?"`
	NotNull bool       `parser:"@(NOTNULL)?"`
}

type FieldExpr struct {
	Pos                lexer.Position
	Name               Ident       `parser:"@Ident"`
	Type               *TypeQName  `parser:"@@"`
	NotNull            bool        `parser:"@(NOTNULL)?"`
	Verifiable         bool        `parser:"@('VERIFIABLE')?"`
	DefaultIntValue    *int        `parser:"('DEFAULT' @Int)?"`
	DefaultStringValue *string     `parser:"('DEFAULT' @String)?"`
	DefaultNextVal     *string     `parser:"(DEFAULTNEXTVAL  '(' @String ')')?"`
	CheckRegexp        *string     `parser:"('CHECK' @String)?"`
	CheckExpression    *Expression `parser:"('CHECK' '(' @@ ')')? "`
}

type ViewStmt struct {
	Statement
	Name     Ident          `parser:"'VIEW' @Ident"`
	Fields   []ViewItemExpr `parser:"'(' @@? (',' @@)* ')'"`
	ResultOf DefQName       `parser:"'AS' 'RESULT' 'OF' @@"`
	pkRef    *PrimaryKeyExpr
}

type ViewItemExpr struct {
	Pos        lexer.Position
	PrimaryKey *PrimaryKeyExpr `parser:"(PRIMARYKEY '(' @@ ')')"`
	Field      *ViewField      `parser:"| @@"`
}

type PrimaryKeyExpr struct {
	PartitionKeyFields      []Ident `parser:"('(' @Ident (',' @Ident)* ')')?"`
	ClusteringColumnsFields []Ident `parser:"(','? @Ident (',' @Ident)*)?"`
}

func (s ViewStmt) GetName() string { return string(s.Name) }

type ViewField struct {
	Name    Ident         `parser:"@Ident"`
	Type    ViewFieldType `parser:"@@"`
	NotNull bool          `parser:"@(NOTNULL)?"`
}

type ViewFieldType struct {
	Int32   bool `parser:"@(('sys' '.')? ('int'|'int32'))"`
	Int64   bool `parser:"| @(('sys' '.')? 'int64')"`
	Float32 bool `parser:"@(('sys' '.')? ('float'|'float32'))"`
	Float64 bool `parser:"| @(('sys' '.')? 'float64')"`
	Blob    bool `parser:"| @('sys.'? 'blob')"`
	Bytes   bool `parser:"| @('sys.'? 'bytes')"`
	Text    bool `parser:"| @('sys.'? 'text')"`
	QName   bool `parser:"| @('sys.'? 'qname')"`
	Bool    bool `parser:"| @('sys.'? 'bool')"`
	Id      bool `parser:"| @(('sys' '.')? 'id')"`
}
