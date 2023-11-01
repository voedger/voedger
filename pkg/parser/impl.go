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
		{Name: "PreStmtComment", Pattern: `(?:(?:[\n\r]+\s*--[^\n]*)+)|(?:[\n\r]\s*\/\*[\s\S]*?\*\/)`},
		{Name: "MultilineComment", Pattern: `\/\*[\s\S]*?\*\/`},
		{Name: "Comment", Pattern: `\s*--[^\r\n]*`},
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
		{Name: "String", Pattern: `('(\\'|[^'])*')`},
		{Name: "Ident", Pattern: `([a-zA-Z_]\w*)|("[a-zA-Z_]\w*")`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
	})

	parser := participle.MustBuild[SchemaAST](
		participle.Lexer(basicLexer),
		participle.Elide("Whitespace", "Comment", "MultilineComment", "PreStmtComment"),
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

func buildPackageSchemaImpl(qualifiedPackageName string, asts []*FileSchemaAST) (*PackageSchemaAST, error) {
	if qualifiedPackageName == "" {
		return nil, ErrNoQualifiedName
	}
	if len(asts) == 0 {
		return nil, nil
	}
	headAst := asts[0].Ast
	for i := 1; i < len(asts); i++ {
		f := asts[i]
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

		if ws, ok := stmt.(*WorkspaceStmt); ok {
			if ws.Descriptor != nil {
				if ws.Descriptor.Name == "" {
					ws.Descriptor.Name = defaultDescriptorName(ws.GetName())
				}
			}
		}

		if named, ok := stmt.(INamedStatement); ok {
			name := named.GetName()
			if _, ok := namedIndex[name]; ok {
				errs = append(errs, errorAt(ErrRedefined(name), named.GetPos()))
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
			var rawComments string
			var mult bool

			if len(s.GetRawCommentBlocks()) < 1 {
				return
			}

			// last block is the statement's comment
			rawComments = s.GetRawCommentBlocks()[len(s.GetRawCommentBlocks())-1]
			rawComments = strings.TrimSpace(rawComments)
			rawComments, mult = strings.CutPrefix(rawComments, "/*")
			if mult {
				rawComments, _ = strings.CutSuffix(rawComments, "*/")
			}

			split := strings.Split(rawComments, "\n")
			fixedComments := make([]string, 0)
			for i := 0; i < len(split); i++ {
				fixed := strings.TrimSpace(split[i])
				if !mult {
					fixed, _ = strings.CutPrefix(fixed, "--")
					fixed = strings.TrimSpace(fixed)
				}
				if len(fixed) > 0 {
					fixedComments = append(fixedComments, fixed)
				}
			}

			s.SetComments(fixedComments)
		}
	})
}

func cleanupImports(schema *SchemaAST) {
	for i := 0; i < len(schema.Imports); i++ {
		imp := &schema.Imports[i]
		imp.Name = strings.Trim(imp.Name, "\"")
	}
}

func defineApp(c *basicContext) {
	var app *ApplicationStmt
	var appAst *PackageSchemaAST

	for _, p := range c.app.Packages {
		a, err := FindApplication(p)
		if err != nil {
			c.errs = append(c.errs, err)
			return
		}
		if a != nil {
			if app != nil {
				c.stmtErr(a.GetPos(), ErrApplicationRedefined)
				return
			}
			app = a
			appAst = p
		}
	}
	if app == nil {
		c.errs = append(c.errs, ErrApplicationNotDefined)
		return
	}

	c.app.Name = string(app.Name)
	appAst.Name = getPackageName(appAst.QualifiedPackageName)
	pkgNames := make(map[string]bool)
	pkgNames[appAst.Name] = true

	for _, use := range app.Uses {

		if _, ok := pkgNames[string(use.Name)]; ok {
			c.stmtErr(use.GetPos(), ErrPackageWithSameNameAlreadyIncludedInApp)
			continue
		}
		pkgNames[string(use.Name)] = true

		pkgQN := GetQualifiedPackageName(use.Name, appAst.Ast)
		if pkgQN == "" {
			c.stmtErr(use.GetPos(), ErrUndefined(string(use.Name)))
			continue
		}

		pkg := c.app.Packages[pkgQN]
		if pkg == nil {
			c.stmtErr(use.GetPos(), ErrCouldNotImport(pkgQN))
			continue
		}

		pkg.Name = string(use.Name)
	}

	for _, p := range c.app.Packages {
		if p.QualifiedPackageName == appdef.SysPackage {
			p.Name = appdef.SysPackage
			continue
		}

		if p.Name == "" {
			c.err(ErrAppDoesNotDefineUseOfPackage(p.QualifiedPackageName))
		}
	}
}

func buildAppSchemaImpl(packages []*PackageSchemaAST) (*AppSchemaAST, error) {

	pkgmap := make(map[string]*PackageSchemaAST)
	for _, p := range packages {
		if _, ok := pkgmap[p.QualifiedPackageName]; ok {
			return nil, ErrPackageRedeclared(p.QualifiedPackageName)
		}
		pkgmap[p.QualifiedPackageName] = p
	}

	appSchema := &AppSchemaAST{
		Packages: pkgmap,
	}

	c := basicContext{
		app:  appSchema,
		errs: make([]error, 0),
	}

	defineApp(&c)
	if len(c.errs) > 0 {
		return nil, errors.Join(c.errs...)
	}

	for _, p := range packages {
		analyse(&c, p)
	}
	return appSchema, errors.Join(c.errs...)
}

type basicContext struct {
	app  *AppSchemaAST
	errs []error
}

func (c *basicContext) newStmtErr(pos *lexer.Position, err error) error {
	return fmt.Errorf("%s: %w", pos.String(), err)
}

func (c *basicContext) stmtErr(pos *lexer.Position, err error) {
	c.err(c.newStmtErr(pos, err))
}

func (c *basicContext) err(err error) {
	c.errs = append(c.errs, err)
}

func buildAppDefs(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder) error {
	ctx := newBuildContext(appSchema, builder)
	return ctx.build()
}
