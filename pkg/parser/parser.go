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
		{Name: "SEMICOLON", Pattern: `;`},
		{Name: "PKGSEPARATOR", Pattern: `\.`},
		{Name: "LEFTBRACKET", Pattern: `\(`},
		{Name: "RIGHTBRACKET", Pattern: `\)`},
		{Name: "ON", Pattern: `ON`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "Int", Pattern: `\d+`},
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
