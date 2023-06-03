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
	errs = analyseDuplicateNames(headAst, errs)
	errs = analyseViews(headAst, errs)
	cleanupComments(headAst)
	cleanupImports(headAst)
	// TODO: unable to specify different base tables (CDOC, WDOC, ...) in the table inheritace chain
	// TODO: Type cannot have nested tables

	return &PackageSchemaAST{
		QualifiedPackageName: qualifiedPackageName,
		Ast:                  headAst,
	}, errors.Join(errs...)
}

func analyseViews(schema *SchemaAST, errs []error) []error {
	iterate(schema, func(stmt interface{}) {
		if view, ok := stmt.(*ViewStmt); ok {
			view.pkRef = nil
			fields := make(map[string]int)
			for i := range view.Fields {
				fe := view.Fields[i]
				if fe.PrimaryKey != nil {
					if view.pkRef != nil {
						errs = append(errs, errorAt(ErrPrimaryKeyRedeclared, &fe.Pos))
					} else {
						view.pkRef = fe.PrimaryKey
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
			if view.pkRef == nil {
				errs = append(errs, errorAt(ErrPrimaryKeyNotDeclared, &view.Pos))
			}
		}
	})
	return errs
}

func analyseDuplicateNames(schema *SchemaAST, errs []error) []error {
	namedIndex := make(map[string]interface{})

	var checkStatement func(stmt interface{})

	checkStatement = func(stmt interface{}) {
		if named, ok := stmt.(INamedStatement); ok {
			name := named.GetName()
			if name == "" {
				_, isProjector := stmt.(*ProjectorStmt)
				if isProjector {
					return // skip anonymous projectors
				}
			}
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
		c.pkg = p
		analyse(&c)
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

func analyse(c *basicContext) {
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
			for _, target := range v.Triggers {
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
	defBuilder interface{}
	qname      appdef.QName
	kind       appdef.DefKind
	names      map[string]bool
}

func (c *defBuildContext) checkName(name string, pos *lexer.Position) error {
	if _, ok := c.names[name]; ok {
		return errorAt(ErrRedeclared(name), pos)
	}
	c.names[name] = true
	return nil
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
	var builder interface{}
	switch kind {
	case appdef.DefKind_CDoc:
		builder = c.builder.AddCDoc(qname)
	case appdef.DefKind_CRecord:
		builder = c.builder.AddCRecord(qname)
	case appdef.DefKind_ODoc:
		builder = c.builder.AddODoc(qname)
	case appdef.DefKind_ORecord:
		builder = c.builder.AddORecord(qname)
	case appdef.DefKind_WDoc:
		builder = c.builder.AddWDoc(qname)
	case appdef.DefKind_WRecord:
		builder = c.builder.AddWRecord(qname)
	case appdef.DefKind_Object:
		builder = c.builder.AddObject(qname)
	default:
		panic(fmt.Sprintf("unsupported def kind %d", kind))
	}
	c.defs = append(c.defs, defBuildContext{
		defBuilder: builder,
		kind:       kind,
		qname:      qname,
		names:      make(map[string]bool),
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

	if err := buildTypes(&ctx); err != nil {
		return err
	}
	if err := buildTables(&ctx); err != nil {
		return err
	}
	if err := buildViews(&ctx); err != nil {
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func buildViews(ctx *buildContext) error {
	for _, schema := range ctx.pkgmap {
		iterateStmt(schema.Ast, func(view *ViewStmt) {
			ctx.newSchema(schema)

			qname := appdef.NewQName(ctx.pkg.Ast.Package, view.Name)
			vb := ctx.builder.AddView(qname)
			for i := range view.Fields {
				f := &view.Fields[i]
				if f.Field != nil {
					datakind := viewFieldDataKind(f.Field)
					if contains(view.pkRef.ClusteringColumnsFields, f.Field.Name) {
						vb.AddClustColumn(f.Field.Name, datakind)
					} else if contains(view.pkRef.PartitionKeyFields, f.Field.Name) {
						vb.AddPartField(f.Field.Name, datakind)
					} else {
						vb.AddValueField(f.Field.Name, datakind, f.Field.NotNull)
					}
				}
			}
		})
	}
	return nil
}

func fillTable(ctx *buildContext, table *TableStmt) {
	if table.Inherits != nil {
		resolve(*table.Inherits, &ctx.basicContext, func(t *TableStmt) error {
			fillTable(ctx, t)
			return nil
		})
	}
	addFieldsOf(table.Of, ctx)
	addTableItems(table.Items, ctx)
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
				fillTable(ctx, table)
				if singletone {
					ctx.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
				}
				ctx.popDef()
			}
		})
	}
	return errors.Join(ctx.errs...)
}

func addFieldRefToDef(refField *RefFieldExpr, ctx *buildContext) {
	if err := ctx.defCtx().checkName(refField.Name, &refField.Pos); err != nil {
		ctx.errs = append(ctx.errs, err)
		return
	}
	refs := make([]appdef.QName, 0)
	errors := false
	for i := range refField.RefDocs {
		_, err := resolveTable(refField.RefDocs[i], &ctx.basicContext, &refField.Pos)
		if err != nil {
			ctx.errs = append(ctx.errs, err)
			errors = true
		}
	}
	if !errors {
		ctx.defCtx().defBuilder.(appdef.IFieldsBuilder).AddRefField(refField.Name, refField.NotNull, refs...)
	}
}

func addFieldToDef(field *FieldExpr, ctx *buildContext) {
	sysDataKind := getTypeDataKind(*field.Type)
	if sysDataKind != appdef.DataKind_null {
		if field.Type.IsArray {
			ctx.errs = append(ctx.errs, errorAt(ErrArrayFieldsNotSupportedHere, &field.Pos))
			return
		}
		if err := ctx.defCtx().checkName(field.Name, &field.Pos); err != nil {
			ctx.errs = append(ctx.errs, err)
			return
		}
		if field.Verifiable {
			// TODO: Support different verification kindsbuilder, &c
			ctx.defCtx().defBuilder.(appdef.IFieldsBuilder).AddVerifiedField(field.Name, sysDataKind, field.NotNull, appdef.VerificationKind_EMail)
		} else {
			ctx.defCtx().defBuilder.(appdef.IFieldsBuilder).AddField(field.Name, sysDataKind, field.NotNull)
		}
	} else {
		// Record?
		pkg := field.Type.Package
		if pkg == "" {
			pkg = ctx.pkg.Ast.Package
		}
		qname := appdef.NewQName(pkg, field.Type.Name)
		wrec := ctx.builder.WRecord(qname)
		crec := ctx.builder.CRecord(qname)
		orec := ctx.builder.ORecord(qname)
		if wrec != nil || orec != nil || crec != nil {
			tk := getNestedTableKind(ctx.defs[0].kind)
			if (wrec != nil && tk != appdef.DefKind_WRecord) ||
				(orec != nil && tk != appdef.DefKind_ORecord) ||
				(crec != nil && tk != appdef.DefKind_CRecord) {
				ctx.errs = append(ctx.errs, ErrNestedTableIncorrectKind)
				return
			}
			ctx.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(field.Name, qname, 0, maxNestedTableContainerOccurrences)
		} else {
			ctx.errs = append(ctx.errs, errorAt(ErrTypeNotSupported(field.Type.String()), &field.Pos))
		}
	}
}

func addConstraintToDef(constraint *TableConstraint, ctx *buildContext) {
	if constraint.UniqueField != nil {
		f := ctx.defCtx().defBuilder.(appdef.IFieldsBuilder).Field(constraint.UniqueField.Field)
		if f == nil {
			ctx.errs = append(ctx.errs, errorAt(ErrUndefinedField(constraint.UniqueField.Field), &constraint.Pos))
			return
		}
		if !f.Required() {
			ctx.errs = append(ctx.errs, errorAt(ErrMustBeNotNull, &constraint.Pos))
			return
		}
		// item.Constraint.ConstraintName  constraint name not used for old uniques
		ctx.defCtx().defBuilder.(appdef.IUniquesBuilder).SetUniqueField(constraint.UniqueField.Field)
	}
}

func addNestedTableToDef(nested *NestedTableStmt, ctx *buildContext) {
	containerName := nested.Name
	if err := ctx.defCtx().checkName(containerName, &nested.Pos); err != nil {
		ctx.errs = append(ctx.errs, err)
		return
	}

	kind, _ := getTableDefKind(&nested.Table, ctx)
	if kind == appdef.DefKind_CDoc || kind == appdef.DefKind_ODoc || kind == appdef.DefKind_WDoc {
		ctx.errs = append(ctx.errs, ErrNestedTableCannotBeDocument)
		return
	}

	tk := getNestedTableKind(ctx.defs[0].kind)
	if kind != appdef.DefKind_null && kind != tk {
		ctx.errs = append(ctx.errs, ErrNestedTableIncorrectKind)
		return
	}

	ctx.pushDef(nested.Table.Name, tk)
	addFieldsOf(nested.Table.Of, ctx)
	addTableItems(nested.Table.Items, ctx)
	ctx.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(containerName, ctx.defCtx().qname, 0, maxNestedTableContainerOccurrences)
	ctx.popDef()

}
func addTableItems(items []TableItemExpr, ctx *buildContext) {
	for _, item := range items {
		if item.RefField != nil {
			addFieldRefToDef(item.RefField, ctx)
		} else if item.Field != nil {
			addFieldToDef(item.Field, ctx)
		} else if item.Constraint != nil {
			addConstraintToDef(item.Constraint, ctx)
		} else if item.NestedTable != nil {
			addNestedTableToDef(item.NestedTable, ctx)
		}
	}
}

func addFieldsOf(types []DefQName, ctx *buildContext) {
	for _, of := range types {
		resolve(of, &ctx.basicContext, func(t *TypeStmt) error {
			addFieldsOf(t.Of, ctx)
			addTableItems(t.Items, ctx)
			return nil
		})
	}
}
