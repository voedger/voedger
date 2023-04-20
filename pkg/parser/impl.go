/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func parse(s string) (*SchemaAST, error) {
	var basicLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "C_SEMICOLON", Pattern: `;`},
		{Name: "C_COMMA", Pattern: `,`},
		{Name: "C_PKGSEPARATOR", Pattern: `\.`},
		{Name: "C_ALL", Pattern: `\*`},
		{Name: "C_EQUAL", Pattern: `=`},
		{Name: "C_LEFTBRACKET", Pattern: `\(`},
		{Name: "C_RIGHTBRACKET", Pattern: `\)`},
		{Name: "C_LEFTSQBRACKET", Pattern: `\[`},
		{Name: "C_RIGHTSQBRACKET", Pattern: `\]`},
		{Name: "ON", Pattern: `ON`},
		{Name: "DEFAULTNEXTVAL", Pattern: `DEFAULT[ \r\n\t]+NEXTVAL`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		{Name: "Int", Pattern: `\d+`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
		{Name: "Comment", Pattern: `--.*`},
	})

	parser := participle.MustBuild[SchemaAST](participle.Lexer(basicLexer), participle.Elide("Whitespace", "Comment"))
	return parser.ParseString("", s)
}

func stringParserImpl(s string) (*SchemaAST, error) {
	parsed, err := parse(s)
	if err != nil {
		return nil, err
	}
	return analyse(parsed)
}

func mergeSchemas(mergeFrom, mergeTo *SchemaAST) {
	// imports
	// TODO: check for import duplicates
	mergeTo.Imports = append(mergeTo.Imports, mergeFrom.Imports...)

	// statements
	mergeTo.Statements = append(mergeTo.Statements, mergeFrom.Statements...)
}

func embedParserImpl(fs embed.FS, dir string) (*SchemaAST, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	schemas := make([]*SchemaAST, 0)
	for _, entry := range entries {
		if strings.ToLower(filepath.Ext(entry.Name())) == ".sql" {
			fp := filepath.Join(dir, entry.Name())
			bytes, err := fs.ReadFile(fp)
			if err != nil {
				return nil, err
			}
			schema, err := parse(string(bytes))
			if err != nil {
				return nil, err
			}
			schemas = append(schemas, schema)
		}
	}
	if len(schemas) == 0 {
		return nil, ErrDirContainsNoSchemaFiles
	}
	head := schemas[0]
	for i := 1; i < len(schemas); i++ {
		schema := schemas[i]
		if schema.Package != head.Package {
			return nil, ErrDirContainsDifferentSchemas
		}
		mergeSchemas(schema, head)
	}
	return analyse(head)
}

func analyse(schema *SchemaAST) (*SchemaAST, error) {

	// TODO: include pos
	namedIndex := make(map[string]int)

	for i := 0; i < len(schema.Statements); i++ {
		var ii interface{} = &schema.Statements[i]

		if statement, ok := ii.(IStatement); ok {
			stmt := statement.Stmt()
			// TODO: recurse into workspaces
			if named, ok := stmt.(INamedStatement); ok {
				if _, ok := namedIndex[named.GetName()]; ok {
					return schema, ErrSchemaContainsDuplicateName(schema.Package, named.GetName())
				}
				namedIndex[named.GetName()] = i
			}
		}

	}
	return schema, nil
}
