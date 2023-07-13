/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"
	fs "io/fs"

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
	Package    string          `parser:"'SCHEMA' @Ident ';'"`
	Imports    []ImportStmt    `parser:"@@? (';' @@)* ';'?"`
	Statements []RootStatement `parser:"@@? (';' @@)* ';'?"`
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
	Name  string  `parser:"'IMPORT' 'SCHEMA' @String"`
	Alias *string `parser:"('AS' @Ident)?"`
}

type RootStatement struct {
	// Only allowed in root
	Template *TemplateStmt `parser:"@@"`

	// Also allowed in root
	Role      *RoleStmt          `parser:"| @@"`
	Comment   *CommentStmt       `parser:"| @@"`
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
	Comment   *CommentStmt            `parser:"| @@"`
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
	Name       string               `parser:"'WORKSPACE' @Ident "`
	Of         []DefQName           `parser:"('OF' @@ (',' @@)*)?"`
	A          int                  `parser:"'('"`
	Descriptor *WsDescriptorStmt    `parser:"('DESCRIPTOR' @@)?"`
	Statements []WorkspaceStatement `parser:"@@? (';' @@)* ';'? ')'"`
}

func (s WorkspaceStmt) GetName() string { return s.Name }
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
	Name  string          `parser:"'TYPE' @Ident "`
	Of    []DefQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
}

func (s TypeStmt) GetName() string { return s.Name }

type WsDescriptorStmt struct {
	Statement
	Of    []DefQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
	_     int             `parser:"';'"`
}

type DefQName struct {
	Package string `parser:"(@Ident '.')?"`
	Name    string `parser:"@Ident"`
}

func (q DefQName) String() string {
	if q.Package == "" {
		return q.Name
	}
	return fmt.Sprintf("%s.%s", q.Package, q.Name)

}

type TypeQName struct {
	Package string `parser:"(@Ident '.')?"`
	Name    string `parser:"@Ident"`
	IsArray bool   `parser:"@Array?"`
}

func (q TypeQName) String() (s string) {
	if q.Package == "" {
		s = q.Name
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
	Comments []string `parser:"@Comment*"`
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
	Name     string       `parser:"'PROJECTOR' @Ident"`
	On       ProjectorOn  `parser:"'ON' @@"`
	Triggers []DefQName   `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	State    []StorageKey `parser:"('STATE'   '(' @@ (',' @@)* ')' )?"`
	Intents  []StorageKey `parser:"('INTENTS' '(' @@ (',' @@)* ')' )?"`
	Engine   EngineType   // Initialized with 1st pass
}

func (s *ProjectorStmt) GetName() string            { return s.Name }
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
	Name      string   `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace DefQName `parser:"@@"`
	Source    string   `parser:"'SOURCE' @Ident"`
}

func (s TemplateStmt) GetName() string { return s.Name }

type RoleStmt struct {
	Statement
	Name string `parser:"'ROLE' @Ident"`
}

func (s RoleStmt) GetName() string { return s.Name }

type TagStmt struct {
	Statement
	Name string `parser:"'TAG' @Ident"`
}

func (s TagStmt) GetName() string { return s.Name }

type CommentStmt struct {
	Statement
	Name  string `parser:"'COMMENT' @Ident"`
	Value string `parser:"@String"`
}

func (s CommentStmt) GetName() string { return s.Name }

type UseTableStmt struct {
	Statement
	Package   string `parser:"'USE' 'TABLE' (@Ident '.')?"`
	Name      string `parser:"(@Ident "`
	AllTables bool   `parser:"| @'*')"`
}

type UseTableItem struct {
	Package   string `parser:"(@Ident '.')?"`
	Name      string `parser:"(@Ident "`
	AllTables bool   `parser:"| @'*')"`
}

/*type sequenceStmt struct {
	Name        string `parser:"'SEQUENCE' @Ident"`
	Type        string `parser:"@Ident"`
	StartWith   *int   `parser:"(('START' 'WITH' @Number)"`
	MinValue    *int   `parser:"| ('MINVALUE' @Number)"`
	MaxValue    *int   `parser:"| ('MAXVALUE' @Number)"`
	IncrementBy *int   `parser:"| ('INCREMENT' 'BY' @Number) )*"`
}*/

type RateStmt struct {
	Statement
	Name   string `parser:"'RATE' @Ident"`
	Amount int    `parser:"@Int"`
	Per    string `parser:"'PER' @('SECOND' | 'MINUTE' | 'HOUR' | 'DAY' | 'YEAR')"`
	PerIP  bool   `parser:"(@('PER' 'IP'))?"`
}

func (s RateStmt) GetName() string { return s.Name }

type GrantStmt struct {
	Statement
	Grants []string `parser:"'GRANT' @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE') (','  @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE'))*"`
	On     string   `parser:"'ON' @('TABLE' | ('ALL' 'TABLES' 'WITH' 'TAG') | 'COMMAND' | ('ALL' 'COMMANDS' 'WITH' 'TAG') | 'QUERY' | ('ALL' 'QUERIES' 'WITH' 'TAG'))"`
	Target DefQName `parser:"@@"`
	To     string   `parser:"'TO' @Ident"`
}

type StorageStmt struct {
	Statement
	Name           string      `parser:"'STORAGE' @Ident"`
	Ops            []StorageOp `parser:"'(' @@ (',' @@)* ')'"`
	RequiresEntity bool        `parser:"@('REQUIRES' 'ENTITY')?"`
}

func (s StorageStmt) GetName() string { return s.Name }

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
	Name    string          `parser:"'FUNCTION' @Ident"`
	Params  []FunctionParam `parser:"'(' @@? (',' @@)* ')'"`
	Returns TypeQName       `parser:"'RETURNS' @@"`
	Engine  EngineType      // Initialized with 1st pass
}

func (s *FunctionStmt) GetName() string            { return s.Name }
func (s *FunctionStmt) SetEngineType(e EngineType) { s.Engine = e }

type CommandStmt struct {
	Statement
	Name        string     `parser:"'COMMAND' @Ident"`
	Arg         *DefQName  `parser:"('(' @@? "`
	UnloggedArg *DefQName  `parser:"(','? UNLOGGED @@)? ')')?"`
	Returns     *DefQName  `parser:"('RETURNS' @@)?"`
	With        []WithItem `parser:"('WITH' @@ (',' @@)* )?"`
	Engine      EngineType // Initialized with 1st pass
}

func (s *CommandStmt) GetName() string            { return s.Name }
func (s *CommandStmt) SetEngineType(e EngineType) { s.Engine = e }

type WithItem struct {
	Comment *DefQName  `parser:"('Comment' '=' @@)"`
	Tags    []DefQName `parser:"| ('Tags' '=' '(' @@ (',' @@)* ')')"`
	Rate    *DefQName  `parser:"| ('Rate' '=' @@)"`
}

type QueryStmt struct {
	Statement
	Name    string     `parser:"'QUERY' @Ident"`
	Arg     *DefQName  `parser:"('(' @@? ')')?"`
	Returns DefQName   `parser:"'RETURNS' @@"`
	With    []WithItem `parser:"('WITH' @@ (',' @@)* )?"`
	Engine  EngineType // Initialized with 1st pass
}

func (s *QueryStmt) GetName() string            { return s.Name }
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
	Name string    `parser:"@Ident"`
	Type TypeQName `parser:"@@"`
}

type TableStmt struct {
	Statement
	Name         string          `parser:"'TABLE' @Ident"`
	Inherits     *DefQName       `parser:"('INHERITS' @@)?"`
	Of           []DefQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items        []TableItemExpr `parser:"'(' @@? (',' @@)* ')'"`
	With         []WithItem      `parser:"('WITH' @@ (',' @@)* )?"`
	tableDefKind appdef.DefKind  // filled on the analysis stage
	singletone   bool
}

func (s TableStmt) GetName() string { return s.Name }

type NestedTableStmt struct {
	Pos   lexer.Position
	Name  string    `parser:"@Ident"`
	Table TableStmt `parser:"@@"`
}

type TableItemExpr struct {
	NestedTable *NestedTableStmt `parser:"@@"`
	Constraint  *TableConstraint `parser:"| @@"`
	RefField    *RefFieldExpr    `parser:"| @@"`
	Field       *FieldExpr       `parser:"| @@"`
}

type TableConstraint struct {
	Pos            lexer.Position
	ConstraintName string           `parser:"('CONSTRAINT' @Ident)?"`
	UniqueField    *UniqueFieldExpr `parser:"(@@"`
	//	Unique         *UniqueExpr      `parser:"(@@"` // TODO: not supported by kernel yet
	Check *TableCheckExpr `parser:"| @@)"`
}

type TableCheckExpr struct {
	Expression Expression `parser:"'CHECK' '(' @@ ')'"`
}

type UniqueFieldExpr struct {
	Field string `parser:"'UNIQUEFIELD' @Ident"`
}

type UniqueExpr struct {
	Fields []string `parser:"'UNIQUE' '(' @Ident (',' @Ident)* ')'"`
}

type RefFieldExpr struct {
	Pos     lexer.Position
	Name    string     `parser:"@Ident"`
	RefDocs []DefQName `parser:"'ref' ('(' @@ (',' @@)* ')')?"`
	NotNull bool       `parser:"@(NOTNULL)?"`
}

type FieldExpr struct {
	Pos                lexer.Position
	Name               string      `parser:"@Ident"`
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
	Name     string         `parser:"'VIEW' @Ident"`
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
	PartitionKeyFields      []string `parser:"('(' @Ident (',' @Ident)* ')')?"`
	ClusteringColumnsFields []string `parser:"','? @Ident (',' @Ident)*"`
}

func (s ViewStmt) GetName() string { return s.Name }

type ViewField struct {
	Name    string        `parser:"@Ident"`
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
