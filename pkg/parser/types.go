/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package sqlschema

type schemaAST struct {
	Package    string          `parser:"'SCHEMA' @Ident SEMICOLON"`
	Imports    []sqlImportStmt `parser:"@@? (';' @@)* SEMICOLON?"`
	Statements []rootStatement `parser:"@@? (';' @@)* SEMICOLON?"`
}

type sqlImportStmt struct {
	Name  string  `parser:"'IMPORT' 'SCHEMA' @String"`
	Alias *string `parser:"('AS' @Ident)?"`
}

type rootStatement struct {
	// Only allowed in root
	Template *templateStmt `parser:"@@"`

	// Also allowed in root
	Role      *roleStmt      `parser:"| @@"`
	Comment   *commentStmt   `parser:"| @@"`
	Tag       *tagStmt       `parser:"| @@"`
	Function  *functionStmt  `parser:"| @@"`
	Workspace *workspaceStmt `parser:"| @@"`
	Table     *tableStmt     `parser:"| @@"`
	Sequence  *sequenceStmt  `parser:"| @@"`
}

type workspaceStatement struct {
	// Only allowed in workspace
	Projector *projectorStmt `parser:"@@"`
	Command   *commandStmt   `parser:"| @@"`
	Query     *queryStmt     `parser:"| @@"`
	Rate      *rateStmt      `parser:"| @@"`
	View      *viewStmt      `parser:"| @@"`
	UseTable  *useTableStmt  `parser:"| @@"`

	// Also allowed in workspace
	Role      *roleStmt      `parser:"| @@"`
	Comment   *commentStmt   `parser:"| @@"`
	Tag       *tagStmt       `parser:"| @@"`
	Function  *functionStmt  `parser:"| @@"`
	Workspace *workspaceStmt `parser:"| @@"`
	Table     *tableStmt     `parser:"| @@"`
	Sequence  *sequenceStmt  `parser:"| @@"`
	Grant     *grantStmt     `parser:"| @@"`
}

type workspaceStmt struct {
	Name       string                `parser:"'WORKSPACE' @Ident '('"`
	Statements []*workspaceStatement `parser:"@@? (SEMICOLON @@)* SEMICOLON? ')'"`
}

type optQName struct {
	Package string `parser:"(@Ident PKGSEPARATOR)?"`
	Name    string `parser:"@Ident"`
}

type projectorStmt struct {
	Name string `parser:"'PROJECTOR' ('ON' | @Ident 'ON')"`
	// TODO
	// On string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | 'INSERT'| 'UPDATE' | 'ACTIVATE'| 'DEACTIVATE' ))"`
	On      string     `parser:"@(('COMMAND' 'ARGUMENT'?) |  'COMMAND' | ('INSERT' ('OR' 'UPDATE')?)  | ('UPDATE' ('OR' 'INSERT')?))"`
	Targets []optQName `parser:"(('IN' '(' @@ (',' @@)* ')') | @@)!"`
	Func    optQName   `parser:"'AS' @@"`
}

type templateStmt struct {
	Name      string   `parser:"'TEMPLATE' @Ident 'OF' 'WORKSPACE'" `
	Workspace optQName `parser:"@@"`
	Source    string   `parser:"'SOURCE' @Ident "`
}

type roleStmt struct {
	Name string `parser:"'ROLE' @Ident"`
}

type tagStmt struct {
	Name string `parser:"'TAG' @Ident"`
}

type commentStmt struct {
	Name  string `parser:"'COMMENT' @Ident"`
	Value string `parser:"@String"`
}

type useTableStmt struct {
	Table useTableItem `parser:"'USE' 'TABLE' @@"`
}

type useTableItem struct {
	Package   string `parser:"(@Ident PKGSEPARATOR)?"`
	Name      string `parser:"(@Ident "`
	AllTables bool   `parser:"| @ALLSYMBOL)"`
}

type sequenceStmt struct {
	Name        string `parser:"'SEQUENCE' @Ident"`
	Type        string `parser:"@Ident"`
	StartWith   *int   `parser:"(('START' 'WITH' @Number)"`
	MinValue    *int   `parser:"| ('MINVALUE' @Number)"`
	MaxValue    *int   `parser:"| ('MAXVALUE' @Number)"`
	IncrementBy *int   `parser:"| ('INCREMENT' 'BY' @Number) )*"`
}

type rateStmt struct {
	Name   string `parser:"'RATE' @Ident"`
	Amount int    `parser:"@Number"`
	Per    string `parser:"'PER' @('SECOND' | 'MINUTE' | 'HOUR' | 'DAY' | 'YEAR')"`
	PerIP  bool   `parser:"(@('PER' 'IP'))?"`
}

type grantStmt struct {
	Grants []string `parser:"'GRANT' @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE') (','  @('ALL' | 'EXECUTE' | 'SELECT' | 'INSERT' | 'UPDATE'))*"`
	On     string   `parser:"'ON' @('TABLE' | ('ALL' 'TABLES' 'WITH' 'TAG') | 'COMMAND' | ('ALL' 'COMMANDS' 'WITH' 'TAG') | 'QUERY' | ('ALL' 'QUERIES' 'WITH' 'TAG'))"`
	Target optQName `parser:"@@"`
	To     string   `parser:"'TO' @Ident"`
}

type functionStmt struct {
	Name    string          `parser:"'FUNCTION' @Ident"`
	Params  []functionParam `parser:"LEFTBRACKET @@? (COMMA @@)* RIGHTBRACKET"`
	Returns optQName        `parser:"'RETURNS' @@"`
	Engine  engineType      `parser:"'ENGINE' @@"`
}

type commandStmt struct {
	Name   string          `parser:"'COMMAND' @Ident"`
	Params []functionParam `parser:"(LEFTBRACKET @@? (',' @@)* RIGHTBRACKET)?"`
	Func   string          `parser:"'AS' @Ident"`
	With   []tcqWithItem   `parser:"('WITH' @@ (COMMA @@)* )?"`
}

type tcqWithItem struct {
	Comment *optQName  `parser:"('Comment' EQUAL @@)"`
	Tags    []optQName `parser:"| ('Tags' EQUAL '[' @@ (COMMA @@)* ']')"`
}

type queryStmt struct {
	Name    string          `parser:"'QUERY' @Ident"`
	Params  []functionParam `parser:"('('@@? (',' @@)* ')')?"`
	Returns optQName        `parser:"'RETURNS' @@"`
	Func    string          `parser:"'AS' @Ident"`
	With    []tcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

type engineType struct {
	WASM    bool `parser:"@'WASM'"`
	Builtin bool `parser:"| @'BUILTIN'"`
}

type functionParam struct {
	NamedParam       *namedParam `parser:"@@"`
	UnnamedParamType *optQName   `parser:"| @@"`
}

type namedParam struct {
	Name string   `parser:"@Ident"`
	Type optQName `parser:"@@"`
}

type tableStmt struct {
	Name  string          `parser:"'TABLE' @Ident"`
	Of    []optQName      `parser:"('OF' @@ (',' @@)*)?"`
	Items []tableItemExpr `parser:"'(' @@ (',' @@)* ')'"`
	With  []tcqWithItem   `parser:"('WITH' @@ (',' @@)* )?"`
}

type tableItemExpr struct {
	Table  *tableStmt  `parser:"@@"`
	Unique *uniqueExpr `parser:"| @@"`
	Field  *fieldExpr  `parser:"| @@"`
}

type uniqueExpr struct {
	Fields []string `parser:"'UNIQUE' @Ident (',' @Ident)*"`
}

// TODO: TABLE: NEXTVAL is unquoted
// TODO: TABLE: FIELD CHECK(expression)
// TODO: TABLE: TABLE CHECK
type fieldExpr struct {
	Name               string    `parser:"@Ident"`
	Type               optQName  `parser:"@@"`
	NotNull            bool      `parser:"@('NOT' 'NULL')?"`
	Verifiable         bool      `parser:"@('VERIFIABLE')?"`
	DefaultIntValue    *int      `parser:"('DEFAULT' @Int)?"`
	DefaultStringValue *string   `parser:"('DEFAULT' @String)?"`
	DefaultNextVal     *string   `parser:"('DEFAULT' 'NEXTVAL' '(' @Ident ')')?"`
	References         *optQName `parser:"('REFERENCES' @@)?"`
	CheckRegexp        *string   `parser:"('CHECK' @String)?"`
}

type viewStmt struct {
	Name     string         `parser:"'VIEW' @Ident"`
	Fields   []viewField    `parser:"'(' @@? (',' @@)* ')'"`
	ResultOf optQName       `parser:"'AS' 'RESULT' 'OF' @@"`
	With     []viewWithItem `parser:"'WITH' @@ (',' @@)* "`
}

type viewField struct {
	Name string `parser:"@Ident"`
	Type string `parser:"@Ident"` // TODO: viewField: predefined types?
}

type viewWithItem struct {
	PrimaryKey *string   `parser:"('PrimaryKey' '=' @String)"`
	Comment    *optQName `parser:"| ('Comment' '=' @@)"`
}
