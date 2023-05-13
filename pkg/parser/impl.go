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
	// TODO: do we need to check that last element in qualifiedPackageName path corresponds to f.Ast.Package?
	for i := 1; i < len(asts); i++ {
		f := asts[i]
		if f.Ast.Package != headAst.Package {
			return nil, ErrUnexpectedSchema(f.FileName, f.Ast.Package, headAst.Package)
		}
		mergeSchemas(f.Ast, headAst)
	}

	errs := make([]error, 0)
	errs = analyseDuplicateNames(headAst, errs)
	errs = analyseDuplicateNamesInSchemas(headAst, errs)
	cleanupComments(headAst)
	cleanupImports(headAst)
	// TODO: unable to specify different base tables (CDOC, WDOC, ...) in the table inheritace chain
	// TODO: Type cannot have nested tables

	return &PackageSchemaAST{
		QualifiedPackageName: qualifiedPackageName,
		Ast:                  headAst,
	}, errors.Join(errs...)
}

func analyseDuplicateNamesInSchemas(schema *SchemaAST, errs []error) []error {
	iterate(schema, func(stmt interface{}) {
		if view, ok := stmt.(*ViewStmt); ok {
			numPK := 0
			fields := make(map[string]int)
			for i := range view.Fields {
				fe := view.Fields[i]
				if fe.PrimaryKey != nil {
					if numPK == 1 {
						errs = append(errs, errorAt(ErrPrimaryKeyRedeclared, &fe.Pos))
					} else {
						numPK++
					}
				}
				if fe.Field != nil {
					if _, ok := fields[fe.Field.Name]; ok {
						errs = append(errs, errorAt(ErrRedeclared(fe.Field.Name), &fe.Pos))
					} else {
						fields[fe.Field.Name] = i
					}
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
		c.pkg = p
		analyseReferences(&c)
	}
	return pkgmap, errors.Join(c.errs...)
}

type basicContext struct {
	pkg    *PackageSchemaAST
	pkgmap map[string]*PackageSchemaAST
	pos    *lexer.Position
	errs   []error
}

func analyzeWithRefs(c *basicContext, with []WithItem) {
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

func analyseReferences(c *basicContext) {
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
			if v.Inherits != nil {
				resolve(*v.Inherits, c, func(f *TableStmt) error { return nil })
			}
			for _, of := range v.Of {
				resolve(of, c, func(f *TypeStmt) error { return nil })
			}
		}
	})
}

type defBuildContext struct {
	defBuilder appdef.IDefBuilder
	qname      appdef.QName
	kind       appdef.DefKind
}

type buildContext struct {
	basicContext
	builder appdef.IAppDefBuilder
	defs    []defBuildContext
}

func (c *buildContext) newSchema(schema *PackageSchemaAST) {
	c.pkg = schema
	c.defs = make([]defBuildContext, 0)
}

func (c *buildContext) pushDef(name string, kind appdef.DefKind) {
	qname := appdef.NewQName(c.pkg.Ast.Package, name)
	c.defs = append(c.defs, defBuildContext{
		defBuilder: c.builder.AddStruct(qname, kind),
		kind:       kind,
		qname:      qname,
	})
}

func (c *buildContext) popDef() {
	c.defs = c.defs[:len(c.defs)-1]
}

func (c *buildContext) defCtx() *defBuildContext {
	return &c.defs[len(c.defs)-1]
}

func newBuildContext(packages map[string]*PackageSchemaAST, builder appdef.IAppDefBuilder) buildContext {
	return buildContext{
		basicContext: basicContext{
			pkg:    nil,
			pkgmap: packages,
			errs:   make([]error, 0),
		},
		builder: builder,
	}
}

func buildAppDefs(packages map[string]*PackageSchemaAST, builder appdef.IAppDefBuilder) error {
	ctx := newBuildContext(packages, builder)

	if err := buildTables(&ctx); err != nil {
		return err
	}
	if err := buildTypes(&ctx); err != nil {
		return err
	}
	return nil
}

func buildTypes(ctx *buildContext) error {
	for _, schema := range ctx.pkgmap {
		iterateStmt(schema.Ast, func(typ *TypeStmt) {
			ctx.newSchema(schema)
			ctx.pushDef(typ.Name, appdef.DefKind_Object)
			addFieldsOf(typ.Of, ctx)
			addTableItems(typ.Items, ctx)
			ctx.popDef()
		})
	}
	return nil
}

func buildTables(ctx *buildContext) error {
	for _, schema := range ctx.pkgmap {
		iterateStmt(schema.Ast, func(table *TableStmt) {
			ctx.newSchema(schema)
			if isPredefinedSysTable(table, ctx) {
				return
			}
			tableType, singletone := getTableDefKind(table, ctx)
			if tableType == appdef.DefKind_null {
				ctx.errs = append(ctx.errs, errorAt(ErrUndefinedTableKind, &table.Pos))
			} else {
				ctx.pushDef(table.Name, tableType)
				addFieldsOf(table.Of, ctx)
				addTableItems(table.Items, ctx)
				if singletone {
					ctx.defCtx().defBuilder.SetSingleton()
				}
				ctx.popDef()
			}
		})
	}
	return errors.Join(ctx.errs...)
}

func addTableItems(items []TableItemExpr, ctx *buildContext) {
	for _, item := range items {
		if item.Field != nil {
			sysDataKind := getTypeDataKind(item.Field.Type)
			if sysDataKind != appdef.DataKind_null {
				if item.Field.Type.IsArray {
					ctx.errs = append(ctx.errs, errorAt(ErrArrayFieldsNotSupportedHere, &item.Field.Pos))
				} else {
					if item.Field.Verifiable {
						// TODO: Support different verification kindsbuilder, &c
						ctx.defCtx().defBuilder.AddVerifiedField(item.Field.Name, sysDataKind, item.Field.NotNull, appdef.VerificationKind_EMail)
					} else {
						ctx.defCtx().defBuilder.AddField(item.Field.Name, sysDataKind, item.Field.NotNull)
					}
				}
			} else {
				ctx.errs = append(ctx.errs, errorAt(ErrTypeNotSupported(item.Field.Type.String()), &item.Field.Pos))
			}
		} else if item.Constraint != nil {
			if item.Constraint.Unique != nil {
				name := item.Constraint.ConstraintName
				if name == "" {
					name = genUniqueName(ctx.defCtx().qname.Entity(), ctx.defCtx().defBuilder)
				}
				ctx.defCtx().defBuilder.AddUnique(name, item.Constraint.Unique.Fields)
			} else if item.Constraint.Check != nil {
				// TODO: implement Table Check COnstraint
			}
		} else if item.Table != nil {
			// Add nested table
			kind, singletone := getTableDefKind(item.Table, ctx)
			if kind != appdef.DefKind_null || singletone {
				ctx.errs = append(ctx.errs, ErrNestedTableCannotBeDocument)
			} else {
				tk := getNestedTableKind(ctx.defs[0].kind) // kind of a root table
				ctx.pushDef(item.Table.Name, tk)
				addFieldsOf(item.Table.Of, ctx)
				addTableItems(item.Table.Items, ctx)
				ctx.defCtx().defBuilder.AddContainer(item.Table.Name, ctx.defCtx().qname, 0, 100) // TODO: max occurances?
				ctx.popDef()
			}
		}
	}
}

func addFieldsOf(types []DefQName, ctx *buildContext) {
	for _, of := range types {
		resolve(of, &ctx.basicContext, func(t *TypeStmt) error {
			addTableItems(t.Items, ctx)
			addFieldsOf(t.Of, ctx)
			return nil
		})
	}
}
