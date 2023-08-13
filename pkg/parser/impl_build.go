/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type buildContext struct {
	basicContext
	builder     appdef.IAppDefBuilder
	defs        []defBuildContext
	wsBuildCtxs map[*WorkspaceStmt]*wsBuildCtx
}

func newBuildContext(packages map[string]*PackageSchemaAST, builder appdef.IAppDefBuilder) *buildContext {
	return &buildContext{
		basicContext: basicContext{
			pkg:    nil,
			pkgmap: packages,
			errs:   make([]error, 0),
		},
		builder:     builder,
		wsBuildCtxs: make(map[*WorkspaceStmt]*wsBuildCtx),
	}
}

type buildFunc func() error

func (c *buildContext) build() error {
	var steps = []buildFunc{
		c.types,
		c.tables,
		c.views,
		c.commands,
		c.queries,
		c.workspaces,
		c.alterWorkspaces,
		c.inheritedWorkspaces,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}

	}
	return errors.Join(c.errs...)
}

type wsBuildCtx struct {
	schema  *PackageSchemaAST
	builder appdef.IWorkspaceBuilder
	qname   appdef.QName
}

func supported(stmt interface{}) bool {
	// FIXME: this must be empty in the end
	if _, ok := stmt.(*TagStmt); ok {
		return false
	}
	if _, ok := stmt.(*RoleStmt); ok {
		return false
	}
	if _, ok := stmt.(*RateStmt); ok {
		return false
	}
	if _, ok := stmt.(*ProjectorStmt); ok {
		return false
	}
	return true
}

func (c *buildContext) useStmtInWs(wsctx *wsBuildCtx, stmtPackage string, stmt interface{}) {
	if named, ok := stmt.(INamedStatement); ok {
		if supported(stmt) {
			wsctx.builder.AddDef(appdef.NewQName(stmtPackage, named.GetName()))
		}
	}
	if useTable, ok := stmt.(*UseTableStmt); ok {
		wsctx.builder.AddDef(appdef.NewQName(stmtPackage, string(useTable.Table)))
	}
	if useWorkspace, ok := stmt.(*UseWorkspaceStmt); ok {
		wsctx.builder.AddDef(appdef.NewQName(stmtPackage, string(useWorkspace.Workspace)))
	}
}

func (c *buildContext) workspaces() error {
	c.wsBuildCtxs = make(map[*WorkspaceStmt]*wsBuildCtx)
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(w *WorkspaceStmt) {
			qname := schema.Ast.NewQName(w.Name)
			bld := c.builder.AddWorkspace(qname)
			c.wsBuildCtxs[w] = &wsBuildCtx{
				schema:  schema,
				qname:   qname,
				builder: bld,
			}
			c.addComments(w, bld)
		})
	}

	var iter func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection)

	iter = func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			c.useStmtInWs(wsctx, string(wsctx.schema.Ast.Package), stmt)
			if collection, ok := stmt.(IStatementCollection); ok {
				if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
					iter(ws, wsctx, collection)
				}
			}
		})
	}

	for w, ctx := range c.wsBuildCtxs {
		c.pkg = ctx.schema
		iter(w, ctx, w)
		if w.Abstract {
			ctx.builder.SetAbstract()
		}
		if w.Descriptor != nil {
			ctx.builder.SetDescriptor(appdef.NewQName(string(ctx.schema.Ast.Package), w.Descriptor.GetName()))
		}
	}
	return nil
}

func (c *buildContext) alterWorkspaces() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(a *AlterWorkspaceStmt) {

			var iter func(wsctx *wsBuildCtx, coll IStatementCollection)
			iter = func(wsctx *wsBuildCtx, coll IStatementCollection) {
				coll.Iterate(func(stmt interface{}) {
					c.useStmtInWs(wsctx, string(schema.Ast.Package), stmt)
					if collection, ok := stmt.(IStatementCollection); ok {
						if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
							iter(wsctx, collection)
						}
					}
				})
			}

			err := resolveEx[*WorkspaceStmt](a.Name, &c.basicContext, func(w *WorkspaceStmt, pkg *PackageSchemaAST) error {
				if !w.Alterable && schema != pkg {
					return ErrWorkspaceIsNotAlterable(w.GetName())
				}
				iter(c.wsBuildCtxs[w], a)
				return nil
			})

			if err != nil {
				c.stmtErr(&a.Pos, err)
			}
		})
	}
	return nil
}

func (c *buildContext) addDefsFromCtx(srcCtx *wsBuildCtx, destBuilder appdef.IWorkspaceBuilder) {
	srcCtx.builder.Defs(func(i appdef.IDef) {
		destBuilder.AddDef(i.QName())
	})
}

func (c *buildContext) inheritedWorkspaces() error {

	sysWorkspace, _, err := lookup[*WorkspaceStmt](DefQName{Package: appdef.SysPackage, Name: rootWorkspaceName}, &c.basicContext)
	if err != nil {
		return ErrSysWorkspaceNotFound
	}

	var addFromInheritedWs func(ws *WorkspaceStmt, wsctx *wsBuildCtx)
	addFromInheritedWs = func(ws *WorkspaceStmt, wsctx *wsBuildCtx) {

		inheritsAnything := false

		for _, inherits := range ws.Inherits {

			inheritsAnything = true
			baseWs, _, err := lookup[*WorkspaceStmt](inherits, &c.basicContext)
			if err != nil {
				c.stmtErr(&ws.Pos, err)
				return
			}
			if baseWs == sysWorkspace {
				c.stmtErr(&ws.Pos, ErrInheritanceFromSysWorkspaceNotAllowed)
				return
			}
			addFromInheritedWs(baseWs, wsctx)
			c.addDefsFromCtx(c.wsBuildCtxs[baseWs], wsctx.builder)
		}

		if !inheritsAnything {
			c.addDefsFromCtx(c.wsBuildCtxs[sysWorkspace], wsctx.builder)
		}
	}

	for w, ctx := range c.wsBuildCtxs {
		c.pkg = ctx.schema
		addFromInheritedWs(w, ctx)
	}
	return nil
}

func (c *buildContext) addComments(s IStatement, builder appdef.ICommentBuilder) {
	comments := s.GetComments()
	if len(comments) > 0 {
		builder.SetComment(comments...)
	}
}

func (c *buildContext) types() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(typ *TypeStmt) {
			c.pushDef(typ.Name, appdef.DefKind_Object)
			c.addComments(typ, c.defCtx().defBuilder.(appdef.ICommentBuilder))
			c.addTableItems(typ.Items)
			c.popDef()
		})
	}
	return nil
}

func (c *buildContext) views() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(view *ViewStmt) {
			qname := c.pkg.Ast.NewQName(view.Name)
			vb := c.builder.AddView(qname)
			c.addComments(view, vb)
			for i := range view.Fields {
				f := &view.Fields[i]
				if f.Field != nil {
					datakind := viewFieldDataKind(f.Field)
					if contains(view.pkRef.ClusteringColumnsFields, f.Field.Name) {
						vb.AddClustColumn(string(f.Field.Name), datakind)
					} else if contains(view.pkRef.PartitionKeyFields, f.Field.Name) {
						vb.AddPartField(string(f.Field.Name), datakind)
					} else {
						vb.AddValueField(string(f.Field.Name), datakind, f.Field.NotNull)
					}
				}
			}
		})
	}
	return nil
}

func (c *buildContext) commands() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(cmd *CommandStmt) {
			qname := c.pkg.Ast.NewQName(cmd.Name)
			b := c.builder.AddCommand(qname)
			c.addComments(cmd, b)
			if cmd.Arg != nil && !isVoid(cmd.Arg.Package, cmd.Arg.Name) {
				argQname := buildQname(c, cmd.Arg.Package, cmd.Arg.Name)
				b.SetArg(argQname)
			}
			if cmd.UnloggedArg != nil && !isVoid(cmd.UnloggedArg.Package, cmd.UnloggedArg.Name) {
				argQname := buildQname(c, cmd.UnloggedArg.Package, cmd.UnloggedArg.Name)
				b.SetUnloggedArg(argQname)
			}
			if cmd.Returns != nil && !isVoid(cmd.Returns.Package, cmd.Returns.Name) {
				retQname := buildQname(c, cmd.Returns.Package, cmd.Returns.Name)
				b.SetResult(retQname)
			}
			if cmd.Engine.WASM {
				b.SetExtension(cmd.GetName(), appdef.ExtensionEngineKind_WASM)
			} else {
				b.SetExtension(cmd.GetName(), appdef.ExtensionEngineKind_BuiltIn)
			}
		})
	}
	return nil
}

func (c *buildContext) queries() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(q *QueryStmt) {
			qname := c.pkg.Ast.NewQName(q.Name)
			b := c.builder.AddQuery(qname)
			c.addComments(q, b)
			if q.Arg != nil && !isVoid(q.Arg.Package, q.Arg.Name) {
				argQname := buildQname(c, q.Arg.Package, q.Arg.Name)
				b.SetArg(argQname)
			}

			if isAny(q.Returns.Package, q.Returns.Name) {
				b.SetResult(istructs.QNameANY)
			} else {
				if !isVoid(q.Returns.Package, q.Returns.Name) {
					retQname := buildQname(c, q.Returns.Package, q.Returns.Name)
					b.SetResult(retQname)
				}
			}

			if q.Engine.WASM {
				b.SetExtension(string(q.Name), appdef.ExtensionEngineKind_WASM)
			} else {
				b.SetExtension(string(q.Name), appdef.ExtensionEngineKind_BuiltIn)
			}
		})
	}
	return nil
}

func (c *buildContext) tables() error {
	for _, schema := range c.pkgmap {
		c.setSchema(schema)
		iterateStmt(schema.Ast, func(table *TableStmt) {
			c.table(schema, table)
		})
		iterateStmt(schema.Ast, func(w *WorkspaceStmt) {
			c.workspaceDescriptor(schema, w)
		})
	}
	return errors.Join(c.errs...)
}

func (c *buildContext) fillTable(table *TableStmt) {
	if table.Inherits != nil {
		if err := resolve(*table.Inherits, &c.basicContext, func(t *TableStmt) error {
			c.fillTable(t)
			return nil
		}); err != nil {
			c.stmtErr(&table.Pos, err)
		}
	}
	c.addTableItems(table.Items)
}

func (c *buildContext) workspaceDescriptor(schema *PackageSchemaAST, w *WorkspaceStmt) {
	if w.Descriptor != nil {
		c.setSchema(schema)
		qname := c.pkg.Ast.NewQName(w.Descriptor.Name)
		if c.isExists(qname, appdef.DefKind_CDoc) {
			return
		}
		c.pushDef(w.Descriptor.Name, appdef.DefKind_CDoc)
		c.addComments(w.Descriptor, c.defCtx().defBuilder.(appdef.ICommentBuilder))
		c.addTableItems(w.Descriptor.Items)
		c.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
		c.popDef()
	}
}

func (c *buildContext) table(schema *PackageSchemaAST, table *TableStmt) {
	c.setSchema(schema)
	if isPredefinedSysTable(c.pkg.QualifiedPackageName, table) {
		return
	}

	qname := c.pkg.Ast.NewQName(table.Name)
	if c.isExists(qname, table.tableDefKind) {
		return
	}
	c.pushDef(table.Name, table.tableDefKind)
	c.addComments(table, c.defCtx().defBuilder.(appdef.ICommentBuilder))
	c.fillTable(table)
	if table.singletone {
		c.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
	}
	if table.Abstract {
		c.defCtx().defBuilder.(appdef.IWithAbstractBuilder).SetAbstract()
	}
	c.popDef()
}

func (c *buildContext) addFieldRefToDef(refField *RefFieldExpr) {
	if err := c.defCtx().checkName(string(refField.Name)); err != nil {
		c.stmtErr(&refField.Pos, err)
		return
	}
	refs := make([]appdef.QName, 0)
	errors := false
	for i := range refField.RefDocs {
		tableStmt, schema, err := resolveTable(refField.RefDocs[i], &c.basicContext)
		if err != nil {
			c.stmtErr(&refField.Pos, err)
			errors = true
			continue
		}
		if err = c.checkReference(refField.RefDocs[i], tableStmt); err != nil {
			c.stmtErr(&refField.Pos, err)
			errors = true
		}
		refs = append(refs, appdef.NewQName(string(schema.Ast.Package), string(refField.RefDocs[i].Name)))
	}
	if !errors {
		c.defCtx().defBuilder.(appdef.IFieldsBuilder).AddRefField(string(refField.Name), refField.NotNull, refs...)
	}
}

func (c *buildContext) addFieldToDef(field *FieldExpr) {
	sysDataKind := getTypeDataKind(*field.Type)
	if sysDataKind != appdef.DataKind_null {
		if field.Type.IsArray {
			c.stmtErr(&field.Pos, ErrArrayFieldsNotSupportedHere)
			return
		}
		if err := c.defCtx().checkName(string(field.Name)); err != nil {
			c.stmtErr(&field.Pos, err)
			return
		}
		if field.Verifiable {
			// TODO: Support different verification kindsbuilder, &c
			c.defCtx().defBuilder.(appdef.IFieldsBuilder).AddVerifiedField(string(field.Name), sysDataKind, field.NotNull, appdef.VerificationKind_EMail)
		} else {
			c.defCtx().defBuilder.(appdef.IFieldsBuilder).AddField(string(field.Name), sysDataKind, field.NotNull)
		}
	} else {
		// Record?
		pkg := field.Type.Package
		if pkg == "" {
			pkg = c.pkg.Ast.Package
		}
		qname := appdef.NewQName(string(pkg), string(field.Type.Name))
		wrec := c.builder.WRecord(qname)
		crec := c.builder.CRecord(qname)
		orec := c.builder.ORecord(qname)

		if wrec == nil && orec == nil && crec == nil { // not yet built
			tbl, _, err := lookup[*TableStmt](DefQName{Package: Ident(qname.Pkg()), Name: Ident(qname.Entity())}, &c.basicContext)
			if err != nil {
				c.stmtErr(&field.Pos, err)
				return
			}
			if tbl == nil {
				c.stmtErr(&field.Pos, ErrTypeNotSupported(field.Type.String()))
				return
			}
			if tbl.Abstract {
				c.stmtErr(&field.Pos, ErrNestedAbstractTable(field.Type.String()))
				return
			}
			if tbl.tableDefKind == appdef.DefKind_CRecord || tbl.tableDefKind == appdef.DefKind_ORecord || tbl.tableDefKind == appdef.DefKind_WRecord {
				c.table(c.pkg, tbl)
				wrec = c.builder.WRecord(qname)
				crec = c.builder.CRecord(qname)
				orec = c.builder.ORecord(qname)
			} else {
				c.stmtErr(&field.Pos, ErrTypeNotSupported(field.Type.String()))
				return
			}
		}

		if wrec != nil || orec != nil || crec != nil {
			//tk := getNestedTableKind(ctx.defs[0].kind)
			tk := getNestedTableKind(c.defCtx().kind)
			if (wrec != nil && tk != appdef.DefKind_WRecord) ||
				(orec != nil && tk != appdef.DefKind_ORecord) ||
				(crec != nil && tk != appdef.DefKind_CRecord) {
				c.errs = append(c.errs, ErrNestedTableIncorrectKind)
				return
			}
			c.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(string(field.Name), qname, 0, maxNestedTableContainerOccurrences)
		} else {
			c.stmtErr(&field.Pos, ErrTypeNotSupported(field.Type.String()))
		}
	}
}

func (c *buildContext) addConstraintToDef(constraint *TableConstraint) {
	if constraint.UniqueField != nil {
		f := c.defCtx().defBuilder.(appdef.IFieldsBuilder).Field(string(constraint.UniqueField.Field))
		if f == nil {
			c.stmtErr(&constraint.Pos, ErrUndefinedField(string(constraint.UniqueField.Field)))
			return
		}
		if !f.Required() {
			c.stmtErr(&constraint.Pos, ErrMustBeNotNull)
			return
		}
		// item.Constraint.ConstraintName  constraint name not used for old uniques
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).SetUniqueField(string(constraint.UniqueField.Field))
	}
}

func (c *buildContext) addNestedTableToDef(nested *NestedTableStmt) {
	nestedTable := &nested.Table
	if nestedTable.tableDefKind == appdef.DefKind_null {
		c.stmtErr(&nestedTable.Pos, ErrUndefinedTableKind)
		return
	}

	containerName := string(nested.Name)
	if err := c.defCtx().checkName(containerName); err != nil {
		c.stmtErr(&nested.Pos, err)
		return
	}

	contQName := c.pkg.Ast.NewQName(nestedTable.Name)
	if !c.isExists(contQName, nestedTable.tableDefKind) {
		c.pushDef(nestedTable.Name, nestedTable.tableDefKind)
		c.addTableItems(nestedTable.Items)
		c.popDef()
	}

	c.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(containerName, contQName, 0, maxNestedTableContainerOccurrences)

}
func (c *buildContext) addTableItems(items []TableItemExpr) {
	for _, item := range items {
		if item.RefField != nil {
			c.addFieldRefToDef(item.RefField)
		} else if item.Field != nil {
			c.addFieldToDef(item.Field)
		} else if item.Constraint != nil {
			c.addConstraintToDef(item.Constraint)
		} else if item.NestedTable != nil {
			c.addNestedTableToDef(item.NestedTable)
		} else if item.FieldSet != nil {
			c.addFieldsOf(&item.FieldSet.Pos, item.FieldSet.Type)
		}
	}
}

func (c *buildContext) addFieldsOf(pos *lexer.Position, of DefQName) {
	if err := resolve(of, &c.basicContext, func(t *TypeStmt) error {
		c.addTableItems(t.Items)
		return nil
	}); err != nil {
		c.stmtErr(pos, err)
	}
}

type defBuildContext struct {
	defBuilder interface{}
	qname      appdef.QName
	kind       appdef.DefKind
	names      map[string]bool
}

func (c *defBuildContext) checkName(name string) error {
	if _, ok := c.names[name]; ok {
		return ErrRedeclared(name)
	}
	c.names[name] = true
	return nil
}

func (c *buildContext) setSchema(schema *PackageSchemaAST) {
	c.pkg = schema
	if c.defs == nil {
		c.defs = make([]defBuildContext, 0)
	}
}

func (c *buildContext) pushDef(name Ident, kind appdef.DefKind) {
	qname := c.pkg.Ast.NewQName(name)
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

func (c *buildContext) isExists(qname appdef.QName, kind appdef.DefKind) (exists bool) {
	switch kind {
	case appdef.DefKind_CDoc:
		return c.builder.CDoc(qname) != nil
	case appdef.DefKind_CRecord:
		return c.builder.CRecord(qname) != nil
	case appdef.DefKind_ODoc:
		return c.builder.ODoc(qname) != nil
	case appdef.DefKind_ORecord:
		return c.builder.ORecord(qname) != nil
	case appdef.DefKind_WDoc:
		return c.builder.WDoc(qname) != nil
	case appdef.DefKind_WRecord:
		return c.builder.WRecord(qname) != nil
	case appdef.DefKind_Object:
		return c.builder.Object(qname) != nil
	default:
		panic(fmt.Sprintf("unsupported def kind %d", kind))
	}
}

func (c *buildContext) fundSchemaByPkg(pkg Ident) *PackageSchemaAST {
	for _, ast := range c.pkgmap {
		if ast.Ast.Package == pkg {
			return ast
		}
	}
	return nil
}

func (c *buildContext) popDef() {
	c.defs = c.defs[:len(c.defs)-1]
}

func (c *buildContext) defCtx() *defBuildContext {
	return &c.defs[len(c.defs)-1]
}

func (c *buildContext) checkReference(refTable DefQName, table *TableStmt) error {
	if refTable.Package == "" {
		refTable.Package = c.basicContext.pkg.Ast.Package
	}
	refTableDef := c.builder.DefByName(appdef.NewQName(string(refTable.Package), string(refTable.Name)))
	if refTableDef == nil {
		c.table(c.fundSchemaByPkg(refTable.Package), table)
		refTableDef = c.builder.DefByName(appdef.NewQName(string(refTable.Package), string(refTable.Name)))
	}

	if refTableDef == nil {
		//if it happened it means that error occurred
		return nil
	}

	for _, defKind := range canNotReferenceTo[c.defCtx().kind] {
		if defKind == refTableDef.Kind() {
			return fmt.Errorf("table %s can not reference to table %s", c.defCtx().qname, refTableDef.QName())
		}
	}

	return nil
}
