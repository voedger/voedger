/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"embed"
	"errors"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func parse(s string) (*SchemaAST, error) {
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

func iterate(c IStatementCollection, callback func(stmt interface{})) {
	c.Iterate(func(stmt interface{}) {
		callback(stmt)
		if collection, ok := stmt.(IStatementCollection); ok {
			iterate(collection, callback)
		}
	})
}

func analyse(schema *SchemaAST) (*SchemaAST, error) {
	errs := make([]error, 0)
	errs = analyseDuplicateNames(schema, errs)
	errs = analyseReferences(schema, errs)
	cleanupComments(schema)
	return schema, errors.Join(errs...)
}

func analyseReferences(schema *SchemaAST, errs []error) []error {
	iterate(schema, func(stmt interface{}) {
		switch v := stmt.(type) {
		case *CommandStmt:
			f := resolveFunc(v.Func, schema)
			if f == nil {
				errs = append(errs, ErrFunctionNotFound(v.Func, v.GetPos()))
			} else {
				errs = CompareParams(v.Params, f, errs)
			}
		case *QueryStmt:
			f := resolveFunc(v.Func, schema)
			if f == nil {
				errs = append(errs, ErrFunctionNotFound(v.Func, v.GetPos()))
			} else {
				errs = CompareParams(v.Params, f, errs)
			}
		case *ProjectorStmt:
			f := resolveFunc(v.Func, schema)
			if f == nil {
				errs = append(errs, ErrFunctionNotFound(v.Func, v.GetPos()))
			} else {
				// TODO: Check funtion params
			}
		}
	})
	return errs
}

func resolveFunc(name OptQName, schema *SchemaAST) (function *FunctionStmt) {
	pkg := strings.TrimSpace(name.Package)
	if pkg == "" || pkg == schema.Package {
		iterate(schema, func(stmt interface{}) {
			if f, ok := stmt.(*FunctionStmt); ok {
				function = f
			}
		})
	} else {
		// TODO: resolve in other packages
	}
	return
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
				errs = append(errs, ErrSchemaContainsDuplicateName(schema.Package, name, s.GetPos()))
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
