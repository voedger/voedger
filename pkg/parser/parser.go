/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func ParseString(s string) (*schemaAST, error) {
	parser := participle.MustBuild[schemaAST]()
	return parser.ParseString("", s)
}

func ParseString2(s string) (*schemaAST, error) {

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
		{Name: "DEFAULT", Pattern: `DEFAULT`},
		{Name: "VERIFIABLE", Pattern: `VERIFIABLE`},
		{Name: "REFERENCES", Pattern: `REFERENCES`},
		{Name: "CHECK", Pattern: `CHECK`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		{Name: "Int", Pattern: `\d+`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
		{Name: "Comment", Pattern: `--.*`},
	})

	parser := participle.MustBuild[schemaAST](participle.Lexer(basicLexer), participle.Elide("Whitespace", "Comment"))
	return parser.ParseString("", s)
}

func ParseDir(dir string) ([]*schemaAST, error) {
	parser := participle.MustBuild[schemaAST]()
	schemas := make([]*schemaAST, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.ToLower(filepath.Ext(path)) == ".sql" {
			fileBytes, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			schema, err := parser.ParseBytes(path, fileBytes)
			if err != nil {
				return err
			}
			schemas = append(schemas, schema)
		}
		return nil
	})

	return schemas, err
}
