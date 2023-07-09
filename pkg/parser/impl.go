/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/voedger/voedger/pkg/appdef"
)

func parseImpl(fileName string, content string) (*SchemaAST, error) {
	var basicLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Comment", Pattern: `--.*`},
		{Name: "Array", Pattern: `\[\]`},
		{Name: "Float", Pattern: `[-+]?\d+\.\d+`},
		{Name: "Int", Pattern: `[-+]?\d+`},
		{Name: "Operators", Pattern: `<>|!=|<=|>=|[-+*/%,()=<>]`}, //( '<>' | '<=' | '>=' | '=' | '<' | '>' | '!=' )"
		{Name: "Punct", Pattern: `[;\[\].]`},
		{Name: "DEFAULTNEXTVAL", Pattern: `DEFAULT[ \r\n\t]+NEXTVAL`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
		{Name: "UNLOGGED", Pattern: `UNLOGGED`},
		{Name: "EXTENSIONENGINE", Pattern: `EXTENSION[ \r\n\t]+ENGINE`},
		{Name: "PRIMARYKEY", Pattern: `PRIMARY[ \r\n\t]+KEY`},
		{Name: "String", Pattern: `("(\\"|[^"])*")|('(\\'|[^'])*')`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
	})

	parser := participle.MustBuild[SchemaAST](
		participle.Lexer(basicLexer),
		participle.Elide("Whitespace", "Comment"),
		participle.Unquote("String"),
	)
	return parser.ParseString(fileName, content)
}

func mergeSchemas(mergeFrom, mergeTo *SchemaAST) {
	// imports
	mergeTo.Imports = append(mergeTo.Imports, mergeFrom.Imports...)

	// statements
	mergeTo.Statements = append(mergeTo.Statements, mergeFrom.Statements...)
}

func parseFSImpl(fs IReadFS, dir string) ([]*FileSchemaAST, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	schemas := make([]*FileSchemaAST, 0)
	for _, entry := range entries {
		if strings.ToLower(filepath.Ext(entry.Name())) == ".sql" {
			var fpath string
			if _, ok := fs.(embed.FS); ok {
				fpath = fmt.Sprintf("%s/%s", dir, entry.Name()) // The path separator is a forward slash, even on Windows systems
			} else {
				fpath = filepath.Join(dir, entry.Name())
			}
			bytes, err := fs.ReadFile(fpath)
			if err != nil {
				return nil, err
			}
			schema, err := parseImpl(entry.Name(), string(bytes))
			if err != nil {
				return nil, err
			}
			schemas = append(schemas, &FileSchemaAST{
				FileName: entry.Name(),
				Ast:      schema,
			})
		}
	}
	if len(schemas) == 0 {
		return nil, ErrDirContainsNoSchemaFiles
	}
	return schemas, nil
}

func mergeFileSchemaASTsImpl(qualifiedPackageName string, asts []*FileSchemaAST) (*PackageSchemaAST, error) {
	if len(asts) == 0 {
		return nil, nil
	}
	headAst := asts[0].Ast
	// TODO: do we need to check that last element in qualifiedPackageName path corresponds to f.Ast.Package?
	for i := 1; i < len(asts); i++ {
		f := asts[i]
		if f.Ast.Package != headAst.Package {
			return nil, ErrUnexpectedSchema(f.FileName, f.Ast.Package, headAst.Package)
		}
		mergeSchemas(f.Ast, headAst)
	}

	errs := make([]error, 0)
	errs = checkDuplicateNames(headAst, errs)
	cleanupComments(headAst)
	cleanupImports(headAst)

	return &PackageSchemaAST{
		QualifiedPackageName: qualifiedPackageName,
		Ast:                  headAst,
	}, errors.Join(errs...)
}

func checkDuplicateNames(schema *SchemaAST, errs []error) []error {
	namedIndex := make(map[string]interface{})

	var checkStatement func(stmt interface{})

	checkStatement = func(stmt interface{}) {
		if named, ok := stmt.(INamedStatement); ok {
			name := named.GetName()
			if _, ok := namedIndex[name]; ok {
				errs = append(errs, errorAt(ErrRedeclared(name), named.GetPos()))
			} else {
				namedIndex[name] = stmt
			}
		}
		if t, ok := stmt.(*TableStmt); ok {
			for i := range t.Items {
				if t.Items[i].NestedTable != nil {
					checkStatement(&t.Items[i].NestedTable.Table)
				}
			}
		}
	}

	iterate(schema, func(stmt interface{}) {
		checkStatement(stmt)
	})
	return errs
}

func cleanupComments(schema *SchemaAST) {
	iterate(schema, func(stmt interface{}) {
		if s, ok := stmt.(IStatement); ok {
			comments := *s.GetComments()
			for i := 0; i < len(comments); i++ {
				comments[i], _ = strings.CutPrefix(comments[i], "--")
				comments[i] = strings.TrimSpace(comments[i])
			}
		}
	})
}

func cleanupImports(schema *SchemaAST) {
	for i := 0; i < len(schema.Imports); i++ {
		imp := &schema.Imports[i]
		imp.Name = strings.Trim(imp.Name, "\"")
	}
}

func mergePackageSchemasImpl(packages []*PackageSchemaAST) (map[string]*PackageSchemaAST, error) {
	pkgmap := make(map[string]*PackageSchemaAST)
	for _, p := range packages {
		if _, ok := pkgmap[p.QualifiedPackageName]; ok {
			return nil, ErrPackageRedeclared(p.QualifiedPackageName)
		}
		pkgmap[p.QualifiedPackageName] = p
	}

	c := basicContext{
		pkg:    nil,
		pkgmap: pkgmap,
		errs:   make([]error, 0),
	}

	for _, p := range packages {
		analyse(&c, p)
	}
	return pkgmap, errors.Join(c.errs...)
}

type basicContext struct {
	pkg    *PackageSchemaAST
	pkgmap map[string]*PackageSchemaAST
	errs   []error
}

func (c *basicContext) stmtErr(pos *lexer.Position, err error) {
	c.errs = append(c.errs, fmt.Errorf("%s: %w", pos.String(), err))
}

func buildAppDefs(packages map[string]*PackageSchemaAST, builder appdef.IAppDefBuilder) error {
	ctx := newBuildContext(packages, builder)
	return ctx.build()
}
