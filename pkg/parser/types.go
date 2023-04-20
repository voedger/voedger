/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package sqlschema

import "embed"

type EmbedParser func(fs embed.FS, dir string) (*SchemaAST, error)
type StringParser func(string) (*SchemaAST, error)

type SchemaAST struct {
	Package    string          `parser:"'SCHEMA' @Ident ';'"`
	Imports    []ImportStmt    `parser:"@@? (';' @@)* ';'?"`
	Statements []RootStatement `parser:"@@? (';' @@)* ';'?"`
}

type ImportStmt struct {
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
}

type WorkspaceStmt struct {
	Comment    *string               `parser:"@Comment?"`
	Name       string                `parser:"'WORKSPACE' @Ident '('"`
	Statements []*WorkspaceStatement `parser:"@@? (C_SEMICOLON @@)* C_SEMICOLON? ')'"`
}

type OptQName struct {
	Package string `parser:"(@Ident C_PKGSEPARATOR)?"`
	Name    string `parser:"@Ident"`
}

type ProjectorStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'PROJECTOR' ('ON' | @Ident 'ON')"`
	// TODO
	// On string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | 'INSERT'| 'UPDATE' | 'ACTIVATE'| 'DEACTIVATE' ))"`
	On      string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | ('INSERT' ('OR' 'UPDATE')?)  | ('UPDATE' ('OR' 'INSERT')?))"`
	Targets []OptQName `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	Func    OptQName   `parser:"'AS' @@"`
}

type TemplateStmt struct {
	Comment   *string  `parser:"@Comment?"`
	Name      string   `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace OptQName `parser:"@@"`
	Source    string   `parser:"'SOURCE' @Ident "`
}

type RoleStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'ROLE' @Ident"`
}

type TagStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'TAG' @Ident"`
}

type CommentStmt struct {
	Comment *string `parser:"@Comment?"`
	Name    string  `parser:"'COMMENT' @Ident"`
	Value   string  `parser:"@String"`
}

type UseTableStmt struct {
	Comment *string      `parser:"@Comment?"`
	Table   UseTableItem `parser:"'USE' 'TABLE' @@"`
}

type UseTableItem struct {
	Package   string `parser:"(@Ident C_PKGSEPARATOR)?"`
	Name      string `parser:"(@Ident "`
	AllTables bool   `parser:"| @C_ALL)"`
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
	Params  []FunctionParam `parser:"C_LEFTBRACKET @@? (C_COMMA @@)* C_RIGHTBRACKET"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Engine  EngineType      `parser:"'ENGINE' @@"`
}

type CommandStmt struct {
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'COMMAND' @Ident"`
	Params  []FunctionParam `parser:"(C_LEFTBRACKET @@? (C_COMMA @@)* C_RIGHTBRACKET)?"`
	Func    string          `parser:"'AS' @Ident"`
	With    []TcqWithItem   `parser:"('WITH' @@ (C_COMMA @@)* )?"`
}

type TcqWithItem struct {
	Comment *OptQName  `parser:"('Comment' C_EQUAL @@)"`
	Tags    []OptQName `parser:"| ('Tags' C_EQUAL C_LEFTSQBRACKET @@ (C_COMMA @@)* C_RIGHTSQBRACKET)"`
}

type QueryStmt struct {
	Comment *string         `parser:"@Comment?"`
	Name    string          `parser:"'QUERY' @Ident"`
	Params  []FunctionParam `parser:"(C_LEFTBRACKET @@? (C_COMMA @@)* C_RIGHTBRACKET)?"`
	Returns OptQName        `parser:"'RETURNS' @@"`
	Func    string          `parser:"'AS' @Ident"`
	With    []TcqWithItem   `parser:"('WITH' @@ (C_COMMA @@)* )?"`
}

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
	Of      []OptQName      `parser:"('OF' @@ (C_COMMA @@)*)?"`
	Items   []TableItemExpr `parser:"C_LEFTBRACKET @@ (C_COMMA @@)* C_RIGHTBRACKET"`
	With    []TcqWithItem   `parser:"('WITH' @@ (C_COMMA @@)* )?"`
}

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
	Verifiable         bool      `parser:"@(VERIFIABLE)?"`
	DefaultIntValue    *int      `parser:"(DEFAULT @Int)?"`
	DefaultStringValue *string   `parser:"(DEFAULT @String)?"`
	DefaultNextVal     *string   `parser:"(DEFAULTNEXTVAL C_LEFTBRACKET @Ident C_RIGHTBRACKET)?"`
	References         *OptQName `parser:"(REFERENCES @@)?"`
	CheckRegexp        *string   `parser:"(CHECK @String)?"`
}

type ViewStmt struct {
	Comment  *string        `parser:"@Comment?"`
	Name     string         `parser:"'VIEW' @Ident"`
	Fields   []ViewField    `parser:"'(' @@? (',' @@)* ')'"`
	ResultOf OptQName       `parser:"'AS' 'RESULT' 'OF' @@"`
	With     []ViewWithItem `parser:"'WITH' @@ (',' @@)* "`
}

type ViewField struct {
	Name string `parser:"@Ident"`
	Type string `parser:"@Ident"` // TODO: viewField: predefined types?
}

type ViewWithItem struct {
	PrimaryKey *string   `parser:"('PrimaryKey' '=' @String)"`
	Comment    *OptQName `parser:"| ('Comment' '=' @@)"`
}
