/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package sqlschema

import (
	"embed"

	"github.com/alecthomas/participle/v2/lexer"
)

type EmbedParser func(fs embed.FS, dir string) (*SchemaAST, error)
type StringParser func(string) (*SchemaAST, error)

type IStatement interface {
	GetPos() *lexer.Position
	GetValue() interface{}
}
type INamedStatement interface {
	GetName() string
}
type IStatementCollection interface {
	Iterate(callback func(stmt IStatement))
}

type statement struct {
	pos   *lexer.Position
	value interface{}
}

func (s *statement) GetPos() *lexer.Position { return s.pos }
func (s *statement) GetValue() interface{}   { return s.value }

type SchemaAST struct {
	Package    string          `parser:"'SCHEMA' @Ident ';'"`
	Imports    []ImportStmt    `parser:"@@? (';' @@)* ';'?"`
	Statements []RootStatement `parser:"@@? (';' @@)* ';'?"`
}

func (s *SchemaAST) Iterate(callback func(stmt IStatement)) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = &statement{
				pos:   &raw.Pos,
				value: extractStatement(*raw),
			}
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
	Pos lexer.Position
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

	stmt IStatement
}

type WorkspaceStatement struct {
	Pos lexer.Position

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

	stmt IStatement
}

type WorkspaceStmt struct {
	Comment    *string              `parser:"@Comment?"`
	Name       string               `parser:"'WORKSPACE' @Ident '('"`
	Statements []WorkspaceStatement `parser:"@@? (';' @@)* ';'? ')'"`
}

func (s WorkspaceStmt) GetName() string { return s.Name }
func (s *WorkspaceStmt) Iterate(callback func(stmt IStatement)) {
	for i := 0; i < len(s.Statements); i++ {
		raw := &s.Statements[i]
		if raw.stmt == nil {
			raw.stmt = &statement{
				pos:   &raw.Pos,
				value: extractStatement(*raw),
			}
		}
		callback(raw.stmt)
	}
}

type OptQName struct {
	Package string `parser:"(@Ident '.')?"`
	Name    string `parser:"@Ident"`
}

type ProjectorStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'PROJECTOR' @Ident? ON"`
	// TODO
	// On string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | 'INSERT'| 'UPDATE' | 'ACTIVATE'| 'DEACTIVATE' ))"`
	On      string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | ('INSERT' ('OR' 'UPDATE')?)  | ('UPDATE' ('OR' 'INSERT')?))"`
	Targets []OptQName `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	Func    OptQName   `parser:"'AS' @@"`
}

func (s ProjectorStmt) GetName() string { return s.Name }

type TemplateStmt struct {
	Comment   *string  `parser:"@Comment?"`
	Name      string   `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace OptQName `parser:"@@"`
	Source    string   `parser:"'SOURCE' @Ident"`
}

func (s TemplateStmt) GetName() string { return s.Name }

type RoleStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'ROLE' @Ident"`
}

func (s RoleStmt) GetName() string { return s.Name }

type TagStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'TAG' @Ident"`
}

func (s TagStmt) GetName() string { return s.Name }

type CommentStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'COMMENT' @Ident"`
	Value   string  `parser:"@String"`
}

func (s CommentStmt) GetName() string { return s.Name }

type UseTableStmt struct {
	Comment *string      `parser:"@Comment?"`
	Table   UseTableItem `parser:"'USE' 'TABLE' @@"`
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
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'RATE' @Ident"`
	Amount  int     `parser:"@Int"`
	Per     string  `parser:"'PER' @('SECOND' | 'MINUTE' | 'HOUR' | 'DAY' | 'YEAR')"`
	PerIP   bool    `parser:"(@('PER' 'IP'))?"`
}

func (s RateStmt) GetName() string { return s.Name }

type GrantStmt struct {
	Comment *string  `parser:"@Comment?"`
	Grants  []string `parser:"'GRANT' @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE') (','  @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE'))*"`
	On      string   `parser:"'ON' @('TABLE' | ('ALL' 'TABLES' 'WITH' 'TAG') | 'COMMAND' | ('ALL' 'COMMANDS' 'WITH' 'TAG') | 'QUERY' | ('ALL' 'QUERIES' 'WITH' 'TAG'))"`
	Target  OptQName `parser:"@@"`
	To      string   `parser:"'TO' @Ident"`
}

type FunctionStmt struct {
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'FUNCTION' @Ident"`
	Params  []FunctionParam `parser:"'(' @@? (',' @@)* ')'"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Engine  EngineType      `parser:"'ENGINE' @@"`
}

func (s FunctionStmt) GetName() string { return s.Name }

type CommandStmt struct {
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'COMMAND' @Ident"`
	Params  []FunctionParam `parser:"('(' @@? (',' @@)* ')')?"`
	Func    string          `parser:"'AS' @Ident"`
	With    []TcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

func (s CommandStmt) GetName() string { return s.Name }

type TcqWithItem struct {
	Comment *OptQName  `parser:"('Comment' '=' @@)"`
	Tags    []OptQName `parser:"| ('Tags' '=' '[' @@ (',' @@)* ']')"`
}

type QueryStmt struct {
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'QUERY' @Ident"`
	Params  []FunctionParam `parser:"('(' @@? (',' @@)* ')')?"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Func    string          `parser:"'AS' @Ident"`
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
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'TABLE' @Ident"`
	Of      []OptQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items   []TableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
	With    []TcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
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

// TODO: TABLE: NEXTVAL is unquoted
// TODO: TABLE: FIELD CHECK(expression)
// TODO: TABLE: TABLE CHECK
type FieldExpr struct {
	Name               string    `parser:"@Ident"`
	Type               OptQName  `parser:"@@"`
	NotNull            bool      `parser:"@(NOTNULL)?"`
	Verifiable         bool      `parser:"@('VERIFIABLE')?"`
	DefaultIntValue    *int      `parser:"('DEFAULT' @Int)?"`
	DefaultStringValue *string   `parser:"('DEFAULT' @String)?"`
	DefaultNextVal     *string   `parser:"(DEFAULTNEXTVAL  '(' @Ident ')')?"`
	References         *OptQName `parser:"('REFERENCES' @@)?"`
	CheckRegexp        *string   `parser:"('CHECK' @String)?"`
}

type ViewStmt struct {
	Comment  *string        `parser:"@Comment?"`
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
