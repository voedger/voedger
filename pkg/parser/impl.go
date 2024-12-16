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
	"github.com/voedger/voedger/pkg/coreutils"
)

func parseImpl(fileName string, content string) (*SchemaAST, error) {
	var basicLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "PreStmtComment", Pattern: `(?:(?:[\n\r]+\s*--[^\n]*)+)|(?:[\n\r]\s*\/\*[\s\S]*?\*\/)`},
		{Name: "MultilineComment", Pattern: `\/\*[\s\S]*?\*\/`},
		{Name: "Comment", Pattern: `\s*--[^\r\n]*`},
		{Name: "Array", Pattern: `\[\]`},
		{Name: "Float", Pattern: `[-+]?\d+\.\d+`},
		{Name: "Int", Pattern: `[-+]?\d+`},
		{Name: "Operators", Pattern: `<>|!=|<=|>=|[-+*/%,()=<>]`}, // ( '<>' | '<=' | '>=' | '=' | '<' | '>' | '!=' )"
		{Name: "Punct", Pattern: `[;\[\].]`},
		{Name: "DEFAULTNEXTVAL", Pattern: `DEFAULT[ \r\n\t]+NEXTVAL`},
		{Name: "NOTNULL", Pattern: `NOT[ \r\n\t]+NULL`},
		{Name: "UNLOGGED", Pattern: `UNLOGGED`},
		{Name: "EXTENSIONENGINE", Pattern: `EXTENSION[ \r\n\t]+ENGINE`},
		{Name: "EXECUTEONCOMMAND", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+COMMAND`},
		{Name: "EXECUTEONALLCOMMANDSWITHTAG", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+COMMANDS[ \r\n\t]+WITH[ \r\n\t]+TAG`},
		{Name: "EXECUTEONALLCOMMANDS", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+COMMANDS`},
		{Name: "EXECUTEONQUERY", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+QUERY`},
		{Name: "EXECUTEONALLQUERIESWITHTAG", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+QUERIES[ \r\n\t]+WITH[ \r\n\t]+TAG`},
		{Name: "EXECUTEONALLQUERIES", Pattern: `EXECUTE[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+QUERIES`},
		{Name: "SELECTONVIEW", Pattern: `SELECT[ \r\n\t]+ON[ \r\n\t]+VIEW`},
		{Name: "SELECTONALLVIEWSWITHTAG", Pattern: `SELECT[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+VIEWS[ \r\n\t]+WITH[ \r\n\t]+TAG`},
		{Name: "SELECTONALLVIEWS", Pattern: `SELECT[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+VIEWS`},
		{Name: "INSERTONWORKSPACE", Pattern: `INSERT[ \r\n\t]+ON[ \r\n\t]+WORKSPACE`},
		{Name: "INSERTONALLWORKSPACESWITHTAG", Pattern: `INSERT[ \r\n\t]+ON[ \r\n\t]+ALL[ \r\n\t]+WORKSPACES[ \r\n\t]+WITH[ \r\n\t]+TAG`},
		{Name: "ONALLTABLESWITHTAG", Pattern: `ON[ \r\n\t]+ALL[ \r\n\t]+TABLES[ \r\n\t]+WITH[ \r\n\t]+TAG`},
		{Name: "ONALLTABLES", Pattern: `ON[ \r\n\t]+ALL[ \r\n\t]+TABLES`},
		{Name: "ONTABLE", Pattern: `ON[ \r\n\t]+TABLE`},
		{Name: "ONVIEW", Pattern: `ON[ \r\n\t]+VIEW`},
		{Name: "SELECT", Pattern: `SELECT`},
		{Name: "TABLE", Pattern: `TABLE`},
		{Name: "PRIMARYKEY", Pattern: `PRIMARY[ \r\n\t]+KEY`},
		{Name: "String", Pattern: `('(\\'|[^'])*')`},
		{Name: "Ident", Pattern: `([a-zA-Z]\w{0,254})|("[a-zA-Z]\w{0,254}")`},
		{Name: "Whitespace", Pattern: `[ \r\n\t]+`},
	})

	parser := participle.MustBuild[SchemaAST](
		participle.Lexer(basicLexer),
		participle.Elide("Whitespace", "Comment", "MultilineComment", "PreStmtComment"),
		participle.Unquote("String"),
		participle.UseLookahead(parserLookahead),
	)
	return parser.ParseString(fileName, content)
}

func mergeSchemas(mergeFrom, mergeTo *SchemaAST) {
	// imports
	mergeTo.Imports = append(mergeTo.Imports, mergeFrom.Imports...)

	// statements
	mergeTo.Statements = append(mergeTo.Statements, mergeFrom.Statements...)
}

func parseFSImpl(fs coreutils.IReadFS, dir string) (schemas []*FileSchemaAST, errs []error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, []error{err}
	}
	for _, entry := range entries {
		fileExt := filepath.Ext(entry.Name())
		if strings.ToLower(fileExt) == VSqlExt || strings.ToLower(fileExt) == SqlExt {
			var fpath string
			if _, ok := fs.(embed.FS); ok {
				if dir == "." || dir == "" {
					fpath = entry.Name()
				} else {
					fpath = fmt.Sprintf("%s/%s", dir, entry.Name()) // The path separator is a forward slash, even on Windows systems
				}
			} else {
				fpath = filepath.Join(dir, entry.Name())
			}
			bytes, err := fs.ReadFile(fpath)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			schema, err := parseImpl(entry.Name(), string(bytes))
			if err != nil {
				errs = append(errs, err)
			} else {
				schemas = append(schemas, &FileSchemaAST{
					FileName: entry.Name(),
					Ast:      schema,
				})
			}
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	if len(schemas) == 0 {
		return nil, []error{ErrDirContainsNoSchemaFiles}
	}
	return schemas, nil
}

func buildPackageSchemaImpl(path string, asts []*FileSchemaAST) (*PackageSchemaAST, error) {
	if path == "" {
		return nil, ErrNoQualifiedName
	}
	if len(asts) == 0 {
		return nil, ErrEmptyFileAstList
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
		Path: path,
		Ast:  headAst,
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
				if t.Items[i].Constraint != nil && t.Items[i].Constraint.ConstraintName != "" {
					checkStatement(t.Items[i].Constraint)
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
	appAst.Name = GetPackageName(appAst.Path)
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

	c.app.LocalNameToFullPath = make(map[string]string, len(c.app.Packages))
	for _, p := range c.app.Packages {
		if p.Path == appdef.SysPackage {
			p.Name = appdef.SysPackage
			continue
		}

		if p.Name == "" {
			c.err(ErrAppDoesNotDefineUseOfPackage(p.Path))
			continue
		}

		c.app.LocalNameToFullPath[p.Name] = p.Path
	}
}

func buildAppSchemaImpl(packages []*PackageSchemaAST) (*AppSchemaAST, error) {

	pkgMap := make(map[string]*PackageSchemaAST)
	pkgPathLocalNames := make(map[string]string, len(packages))
	var importErrors []error
	for _, p := range packages {
		if _, ok := pkgMap[p.Path]; ok {
			return nil, ErrPackageRedeclared(p.Path)
		}
		// check for local package name redeclaration
		for _, imp := range p.Ast.Imports {
			localPkgName, ok := pkgPathLocalNames[imp.Name]
			if !ok {
				if imp.Alias != nil {
					pkgPathLocalNames[imp.Name] = string(*imp.Alias)
				} else {
					pkgPathLocalNames[imp.Name] = filepath.Base(imp.Name)
				}
				continue
			}

			newLocalPkgName := filepath.Base(imp.Name)
			if imp.Alias != nil {
				newLocalPkgName = string(*imp.Alias)
			}
			if newLocalPkgName != localPkgName {
				importErrors = append(importErrors, fmt.Errorf("%s: %w", imp.Pos.String(), ErrLocalPackageNameRedeclared(localPkgName, newLocalPkgName)))
			}
		}
		pkgMap[p.Path] = p
	}

	appSchema := &AppSchemaAST{
		Packages: pkgMap,
	}

	c := basicContext{
		app:  appSchema,
		errs: make([]error, 0),
	}

	c.errs = append(c.errs, importErrors...)

	defineApp(&c)

	preAnalyse(&c, packages)
	if len(c.errs) > 0 {
		return nil, errors.Join(c.errs...)
	}

	analyse(&c, packages)

	if len(c.errs) > 0 {
		return nil, errors.Join(c.errs...)
	}
	return appSchema, nil
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

func buildAppDefs(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder, opts ...BuildAppDefsOption) error {
	ctx := newBuildContext(appSchema, builder)
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx.build()
}
