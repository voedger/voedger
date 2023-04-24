/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package sqlschema

import (
	"embed"
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

// TODO: why embed.FS, how to process a normal folder?
// NewFSParser()
type EmbedParser func(fs embed.FS, dir string) (*SchemaAST, error)

type StringParser func(string) (*SchemaAST, error)

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
	Role      *RoleStmt      `parser:"| @@"`
	Comment   *CommentStmt   `parser:"| @@"`
	Tag       *TagStmt       `parser:"| @@"`
	Function  *FunctionStmt  `parser:"| @@"`
	Workspace *WorkspaceStmt `parser:"| @@"`
	Table     *TableStmt     `parser:"| @@"`
	// Sequence  *sequenceStmt  `parser:"| @@"`

	stmt interface{}
}

type WorkspaceStatement struct {
	// Only allowed in workspace
	Projector *ProjectorStmt `parser:"@@"`
	Command   *CommandStmt   `parser:"| @@"`
	Query     *QueryStmt     `parser:"| @@"`
	Rate      *RateStmt      `parser:"| @@"`
	View      *ViewStmt      `parser:"| @@"`
	UseTable  *UseTableStmt  `parser:"| @@"`

	// Also allowed in workspace
	Role      *RoleStmt      `parser:"| @@"`
	Comment   *CommentStmt   `parser:"| @@"`
	Tag       *TagStmt       `parser:"| @@"`
	Function  *FunctionStmt  `parser:"| @@"`
	Workspace *WorkspaceStmt `parser:"| @@"`
	Table     *TableStmt     `parser:"| @@"`
	//Sequence  *sequenceStmt  `parser:"| @@"`
	Grant *GrantStmt `parser:"| @@"`

	stmt interface{}
}

type WorkspaceStmt struct {
	Statement
	Name       string               `parser:"'WORKSPACE' @Ident '('"`
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

type OptQName struct {
	Package string `parser:"(@Ident '.')?"`
	Name    string `parser:"@Ident"`
}

func (q OptQName) String() string {
	if q.Package == "" {
		return q.Name
	}
	return fmt.Sprintf("%s.%s", q.Package, q.Name)

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

type ProjectorStmt struct {
	Statement
	Name string `parser:"'PROJECTOR' @Ident? 'ON'"`
	// TODO
	// On string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | 'INSERT'| 'UPDATE' | 'ACTIVATE'| 'DEACTIVATE' ))"`
	On      string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | ('INSERT' ('OR' 'UPDATE')?)  | ('UPDATE' ('OR' 'INSERT')?))"`
	Targets []OptQName `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	Func    OptQName   `parser:"'AS' @@"`
}

func (s ProjectorStmt) GetName() string { return s.Name }

type TemplateStmt struct {
	Statement
	Name      string   `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace OptQName `parser:"@@"`
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
	Target OptQName `parser:"@@"`
	To     string   `parser:"'TO' @Ident"`
}

type FunctionStmt struct {
	Statement
	Name    string          `parser:"'FUNCTION' @Ident"`
	Params  []FunctionParam `parser:"'(' @@? (',' @@)* ')'"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Engine  EngineType      `parser:"'ENGINE' @@"`
}

func (s FunctionStmt) GetName() string { return s.Name }

type CommandStmt struct {
	Statement
	Name   string          `parser:"'COMMAND' @Ident"`
	Params []FunctionParam `parser:"('(' @@? (',' @@)* ')')?"`
	Func   OptQName        `parser:"'AS' @@"`
	With   []TcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

func (s CommandStmt) GetName() string { return s.Name }

type TcqWithItem struct {
	Comment *OptQName  `parser:"('Comment' '=' @@)"`
	Tags    []OptQName `parser:"| ('Tags' '=' '[' @@ (',' @@)* ']')"`
}

type QueryStmt struct {
	Statement
	Name    string          `parser:"'QUERY' @Ident"`
	Params  []FunctionParam `parser:"('(' @@? (',' @@)* ')')?"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Func    OptQName        `parser:"'AS' @@"`
	With    []TcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

func (s QueryStmt) GetName() string { return s.Name }

type EngineType struct {
	WASM    bool `parser:"@'WASM'"`
	Builtin bool `parser:"| @'BUILTIN'"`
}

type FunctionParam struct {
	NamedParam       *NamedParam `parser:"@@"`
	UnnamedParamType *OptQName   `parser:"| @@"`
}

type NamedParam struct {
	Name string   `parser:"@Ident"`
	Type OptQName `parser:"@@"`
}

type TableStmt struct {
	Statement
	Name  string          `parser:"'TABLE' @Ident"`
	Of    []OptQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
	With  []TcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

func (s TableStmt) GetName() string { return s.Name }

type TableItemExpr struct {
	Table  *TableStmt  `parser:"@@"`
	Unique *UniqueExpr `parser:"| @@"`
	Field  *FieldExpr  `parser:"| @@"`
}

type UniqueExpr struct {
	Fields []string `parser:"'UNIQUE' @Ident (',' @Ident)*"`
}

// TODO: TABLE: FIELD CHECK(expression)
// TODO: TABLE: TABLE CHECK
type FieldExpr struct {
	Name               string    `parser:"@Ident"`
	Type               OptQName  `parser:"@@"`
	NotNull            bool      `parser:"@(NOTNULL)?"`
	Verifiable         bool      `parser:"@('VERIFIABLE')?"`
	DefaultIntValue    *int      `parser:"('DEFAULT' @Int)?"`
	DefaultStringValue *string   `parser:"('DEFAULT' @String)?"`
	DefaultNextVal     *string   `parser:"(DEFAULTNEXTVAL  '(' @String ')')?"`
	References         *OptQName `parser:"('REFERENCES' @@)?"`
	CheckRegexp        *string   `parser:"('CHECK' @String)?"`
}

type ViewStmt struct {
	Statement
	Name     string         `parser:"'VIEW' @Ident"`
	Fields   []ViewField    `parser:"'(' @@? (',' @@)* ')'"`
	ResultOf OptQName       `parser:"'AS' 'RESULT' 'OF' @@"`
	With     []ViewWithItem `parser:"'WITH' @@ (',' @@)* "`
}

func (s ViewStmt) GetName() string { return s.Name }

type ViewField struct {
	Name string `parser:"@Ident"`
	Type string `parser:"@Ident"` // TODO: viewField: predefined types?
}

type ViewWithItem struct {
	PrimaryKey *string   `parser:"('PrimaryKey' '=' @String)"`
	Comment    *OptQName `parser:"| ('Comment' '=' @@)"`
}
