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

		{Name: "Punct", Pattern: `(;|,|\.|\*|=|\(|\)|\[|\])`},
		{Name: "Keywords", Pattern: `ON`},
		{Name: "DEFAULTNEXTVAL", Pattern: `DEFAULT[ \r\n\t]+NEXTVAL`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
		{Name: "PRIMARYKEY", Pattern: `PRIMARY[ \r\n\t]+KEY`},
		{Name: "String", Pattern: `("(\\"|[^"])*")|('(\\'|[^'])*')`},
		{Name: "Int", Pattern: `\d+`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
		{Name: "Comment", Pattern: `--.*`},
	})

	parser := participle.MustBuild[SchemaAST](participle.Lexer(basicLexer), participle.Elide("Whitespace", "Comment"))
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
	errs := make([]error, 0)
	pkgmap := make(map[string]*PackageSchemaAST)
	for _, p := range packages {
		if _, ok := pkgmap[p.QualifiedPackageName]; ok {
			return ErrPackageRedeclared(p.QualifiedPackageName)
		}
		pkgmap[p.QualifiedPackageName] = p
	}

	for _, p := range packages {
		errs = analyseReferences(p, pkgmap, errs)
	}
	return errors.Join(errs...)
}

func analyseReferences(pkg *PackageSchemaAST, pkgmap map[string]*PackageSchemaAST, errs []error) []error {
	iterate(pkg.Ast, func(stmt interface{}) {
		var err error
		var pos *lexer.Position
		switch v := stmt.(type) {
		case *CommandStmt:
			pos = &v.Pos
			err = resolveFunc(v.Func, pkg, pkgmap, func(f *FunctionStmt) error {
				return CompareParams(v.Params, f)
			})
		case *QueryStmt:
			pos = &v.Pos
			err = resolveFunc(v.Func, pkg, pkgmap, func(f *FunctionStmt) error {
				err = CompareParams(v.Params, f)
				if err != nil {
					return err
				}
				if v.Returns != f.Returns {
					return ErrFunctionResultIncorrect
				}
				return nil
			})
		case *ProjectorStmt:
			pos = &v.Pos
			err = resolveFunc(v.Func, pkg, pkgmap, func(f *FunctionStmt) error {
				// TODO: Check function params
				// TODO: Check ON (Command, Argument type, CUD)

				return nil
			})
		}
		if err != nil {
			errs = append(errs, errorAt(err, pos))
		}
	})
	return errs
}
