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
	errs = analyseInernalReferences(headAst, errs)
	cleanupComments(headAst)

	return &PackageSchemaAST{
		QualifiedPackageName: qualifiedPackageName,
		Ast:                  headAst,
	}, errors.Join(errs...)
}

func analyseInernalReferences(schema *SchemaAST, errs []error) []error {
	iterate(schema, func(stmt interface{}) {
		switch v := stmt.(type) {
		case *CommandStmt:
			if isInternalFunc(v.Func, schema) {
				f := resolveFunc(v.Func.Name, schema)
				if f == nil {
					errs = append(errs, errorAt(ErrUndefined(v.Func.String()), v.GetPos()))
				} else {
					errs = CompareParams(&v.Pos, v.Params, f, errs)
				}
			}
		case *QueryStmt:
			if isInternalFunc(v.Func, schema) {
				f := resolveFunc(v.Func.Name, schema)
				if f == nil {
					errs = append(errs, errorAt(ErrUndefined(v.Func.String()), v.GetPos()))
				} else {
					errs = CompareParams(&v.Pos, v.Params, f, errs)
					if v.Returns != f.Returns {
						errs = append(errs, errorAt(ErrFunctionResultIncorrect, v.GetPos()))
					}
				}
			}
		case *ProjectorStmt:
			if isInternalFunc(v.Func, schema) {
				f := resolveFunc(v.Func.Name, schema)
				if f == nil {
					errs = append(errs, errorAt(ErrUndefined(v.Func.String()), v.GetPos()))
				} else {
					// TODO: Check function params
					// TODO: Check ON (Command, Argument type, CUD)
				}
			}
		}
	})
	return errs
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
		errs = analyseExternalReferences(p, pkgmap, errs)
	}
	return nil
}

func analyseExternalReferences(pkg *PackageSchemaAST, pkgmap map[string]*PackageSchemaAST, errs []error) []error {
	iterate(pkg.Ast, func(stmt interface{}) {
		switch v := stmt.(type) {
		case *CommandStmt:
			if !isInternalFunc(v.Func, pkg.Ast) {
				// f := resolveFunc(v.Func.Name, schema)
				// if f == nil {
				// 	errs = append(errs, errorAt(ErrUndefined(v.Func.String()), v.GetPos()))
				// } else {
				// 	errs = CompareParams(&v.Pos, v.Params, f, errs)
				// }
			}
		case *QueryStmt:
			//
		case *ProjectorStmt:
			//
		}
	})
	return errs
}
