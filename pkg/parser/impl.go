/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func parseImpl(fileName string, content string) (*SchemaAST, error) {
	var basicLexer = lexer.MustSimple([]lexer.SimpleRule{

		{Name: "Comment", Pattern: `--.*`},
		{Name: "Array", Pattern: `\[\]`},
		{Name: "Float", Pattern: `[-+]?\d+\.\d+`},
		{Name: "Int", Pattern: `[-+]?\d+`},
		{Name: "Operators", Pattern: `<>|!=|<=|>=|[-+*/%,()=<>]`}, //( '<>' | '<=' | '>=' | '=' | '<' | '>' | '!=' )"
		{Name: "Punct", Pattern: `[;\[\].]`},
		{Name: "Keywords", Pattern: `ON|AND|OR`},
		{Name: "DEFAULTNEXTVAL", Pattern: `DEFAULT[ \r\n\t]+NEXTVAL`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
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
			fp := filepath.Join(dir, entry.Name())
			bytes, err := fs.ReadFile(fp)
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

	for i := 1; i < len(asts); i++ {
		f := asts[i]
		if f.Ast.Package != headAst.Package {
			return nil, ErrUnexpectedSchema(f.FileName, f.Ast.Package, headAst.Package)
		}
		mergeSchemas(f.Ast, headAst)
	}

	errs := make([]error, 0)
	errs = analyseDuplicateNames(headAst, errs)
	cleanupComments(headAst)
	cleanupImports(headAst)

	return &PackageSchemaAST{
		QualifiedPackageName: qualifiedPackageName,
		Ast:                  headAst,
	}, errors.Join(errs...)
}

func analyseDuplicateNames(schema *SchemaAST, errs []error) []error {
	namedIndex := make(map[string]interface{})

	iterate(schema, func(stmt interface{}) {
		if named, ok := stmt.(INamedStatement); ok {
			name := named.GetName()
			if name == "" {
				_, isProjector := stmt.(*ProjectorStmt)
				if isProjector {
					return // skip anonymous projectors
				}
			}
			if _, ok := namedIndex[name]; ok {
				s := stmt.(IStatement)
				errs = append(errs, errorAt(ErrRedeclared(name), s.GetPos()))
			} else {
				namedIndex[name] = stmt
			}
		}
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

func mergePackageSchemasImpl(packages []*PackageSchemaAST) error {
	pkgmap := make(map[string]*PackageSchemaAST)
	for _, p := range packages {
		if _, ok := pkgmap[p.QualifiedPackageName]; ok {
			return ErrPackageRedeclared(p.QualifiedPackageName)
		}
		pkgmap[p.QualifiedPackageName] = p
	}

	c := aContext{
		pkg:    nil,
		pkgmap: pkgmap,
		errs:   make([]error, 0),
	}

	for _, p := range packages {
		c.pkg = p
		analyseReferences(&c)
	}
	return errors.Join(c.errs...)
}

type aContext struct {
	pkg    *PackageSchemaAST
	pkgmap map[string]*PackageSchemaAST
	pos    *lexer.Position
	errs   []error
}

func analyzeWithRefs(c *aContext, with []WithItem) {
	for i := range with {
		wi := &with[i]
		if wi.Comment != nil {
			resolve(*wi.Comment, c, func(f *CommentStmt) error { return nil })
		} else if wi.Rate != nil {
			resolve(*wi.Rate, c, func(f *RateStmt) error { return nil })
		}
		for j := range wi.Tags {
			tag := wi.Tags[j]
			resolve(tag, c, func(f *TagStmt) error { return nil })
		}
	}
}

func analyseReferences(c *aContext) {
	iterate(c.pkg.Ast, func(stmt interface{}) {
		switch v := stmt.(type) {
		case *CommandStmt:
			c.pos = &v.Pos
			analyzeWithRefs(c, v.With)
		case *QueryStmt:
			c.pos = &v.Pos
			analyzeWithRefs(c, v.With)
		case *ProjectorStmt:
			c.pos = &v.Pos
			// Check targets
			for _, target := range v.Targets {
				if v.On.Activate || v.On.Deactivate || v.On.Insert || v.On.Update {
					resolve(target, c, func(f *TableStmt) error { return nil })
				} else if v.On.Command {
					resolve(target, c, func(f *CommandStmt) error { return nil })
				} else if v.On.CommandArgument {
					resolve(target, c, func(f *TypeStmt) error { return nil })
				}
			}
		case *TableStmt:
			c.pos = &v.Pos
			analyzeWithRefs(c, v.With)
		}
	})
}
