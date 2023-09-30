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

func newBuildContext(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder) *buildContext {
	return &buildContext{
		basicContext: basicContext{
			app:  appSchema,
			errs: make([]error, 0),
		},
		builder:     builder,
		wsBuildCtxs: make(map[*WorkspaceStmt]*wsBuildCtx),
		defs:        make([]defBuildContext, 0),
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
	pkg     *PackageSchemaAST
	builder appdef.IWorkspaceBuilder
	ictx    *iterateCtx
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

func (c *buildContext) useStmtInWs(wsctx *wsBuildCtx, stmtPackage string, stmt interface{}, ictx *iterateCtx) {
	if named, ok := stmt.(INamedStatement); ok {
		if supported(stmt) {
			wsctx.builder.AddType(appdef.NewQName(stmtPackage, named.GetName()))
		}
	}
	if useTable, ok := stmt.(*UseTableStmt); ok {
		err := resolveInCtx(useTable.Table, ictx, func(tbl *TableStmt, pkg *PackageSchemaAST) error {
			wsctx.builder.AddType(pkg.NewQName(tbl.Name))
			return nil
		})
		if err != nil {
			// notest
			c.stmtErr(&useTable.Pos, err)
			return
		}
	}
	if useWorkspace, ok := stmt.(*UseWorkspaceStmt); ok {
		wsctx.builder.AddType(appdef.NewQName(stmtPackage, string(useWorkspace.Workspace)))
	}
}

func (c *buildContext) workspaces() error {

	var iter func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection)

	iter = func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			c.useStmtInWs(wsctx, wsctx.pkg.Name, stmt, wsctx.ictx)
			if collection, ok := stmt.(IStatementCollection); ok {
				if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
					iter(ws, wsctx, collection)
				}
			}
		})
	}

	c.wsBuildCtxs = make(map[*WorkspaceStmt]*wsBuildCtx)
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			qname := schema.NewQName(w.Name)
			bld := c.builder.AddWorkspace(qname)
			wsc := &wsBuildCtx{
				pkg:     schema,
				qname:   qname,
				builder: bld,
				ictx:    ictx,
			}
			c.wsBuildCtxs[w] = wsc
			c.addComments(w, bld)
		})
	}

	for w, wsc := range c.wsBuildCtxs {
		iter(w, wsc, w)
		if w.Abstract {
			wsc.builder.SetAbstract()
		}
		if w.Descriptor != nil {
			wsc.builder.SetDescriptor(appdef.NewQName(string(wsc.ictx.pkg.Name), w.Descriptor.GetName()))
		}

	}

	return nil
}

func (c *buildContext) alterWorkspaces() error {
	for _, pkgAst := range c.app.Packages {
		iteratePackageStmt(pkgAst, &c.basicContext, func(a *AlterWorkspaceStmt, ictx *iterateCtx) {
			var iter func(wsctx *wsBuildCtx, coll IStatementCollection)
			iter = func(wsctx *wsBuildCtx, coll IStatementCollection) {
				coll.Iterate(func(stmt interface{}) {
					c.useStmtInWs(wsctx, string(pkgAst.Name), stmt, ictx)
					if collection, ok := stmt.(IStatementCollection); ok {
						if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
							iter(wsctx, collection)
						}
					}
				})
			}

			err := resolveInCtx(a.Name, ictx, func(w *WorkspaceStmt, pkg *PackageSchemaAST) error {
				if !w.Alterable && pkg != pkgAst {
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
	srcCtx.builder.Types(func(t appdef.IType) {
		destBuilder.AddType(t.QName())
	})
}

func (c *buildContext) inheritedWorkspaces() error {
	sysWorkspace, err := lookupInSysPackage(&c.basicContext, DefQName{Package: appdef.SysPackage, Name: rootWorkspaceName})
	if err != nil {
		return err
	}

	var addFromInheritedWs func(ws *WorkspaceStmt, wsctx *wsBuildCtx)
	addFromInheritedWs = func(ws *WorkspaceStmt, wsctx *wsBuildCtx) {

		inheritsAnything := false

		for _, inherits := range ws.Inherits {

			inheritsAnything = true
			baseWs, _, err := lookupInCtx[*WorkspaceStmt](inherits, wsctx.ictx)
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
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(typ *TypeStmt, ictx *iterateCtx) {
			c.pushDef(schema.NewQName(typ.Name), appdef.TypeKind_Object)
			c.addComments(typ, c.defCtx().defBuilder.(appdef.ICommentBuilder))
			c.addTableItems(typ.Items, ictx)
			c.popDef()
		})
	}
	return nil
}

func (c *buildContext) views() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(view *ViewStmt, ictx *iterateCtx) {
			c.pushDef(schema.NewQName(view.Name), appdef.TypeKind_ViewRecord)
			vb := func() appdef.IViewBuilder {
				return c.defCtx().defBuilder.(appdef.IViewBuilder)
			}
			c.addComments(view, vb())
			for i := range view.Items {
				f := &view.Items[i]

				if f.PrimaryKey != nil {
					continue
				}

				ccComments := func(fieldname string, comments []string) {
					if len(comments) > 0 {
						vb().Key().ClustCols().SetFieldComment(string(fieldname), comments...)
					}
				}
				pkComments := func(fieldname string, comments []string) {
					if len(comments) > 0 {
						vb().Key().Partition().SetFieldComment(string(fieldname), comments...)
					}
				}
				valComments := func(fieldname string, comments []string) {
					if len(comments) > 0 {
						vb().Value().SetFieldComment(string(fieldname), comments...)
					}
				}

				if f.Field != nil {
					fieldname := f.Field.Name
					comments := f.Field.Statement.GetComments()
					var length uint16 = appdef.DefaultFieldMaxLength
					if f.Field.Type.Varchar != nil {
						if f.Field.Type.Varchar.MaxLen != nil {
							length = *f.Field.Type.Varchar.MaxLen
						}
						if contains(view.pkRef.ClusteringColumnsFields, fieldname) {
							vb().Key().ClustCols().AddStringField(string(fieldname), length)
							ccComments(string(fieldname), comments)
						} else {
							vb().Value().AddStringField(string(fieldname), f.Field.NotNull, length)
							valComments(string(fieldname), comments)
						}
					} else if f.Field.Type.Bytes != nil {
						if f.Field.Type.Bytes.MaxLen != nil {
							length = *f.Field.Type.Bytes.MaxLen
						}
						if contains(view.pkRef.ClusteringColumnsFields, fieldname) {
							vb().Key().ClustCols().AddBytesField(string(fieldname), length)
							ccComments(string(fieldname), comments)
						} else {
							vb().Value().AddBytesField(string(fieldname), f.Field.NotNull, length)
							valComments(string(fieldname), comments)
						}
					} else { // Other data types
						datakind := dataTypeToDataKind(f.Field.Type)
						if contains(view.pkRef.ClusteringColumnsFields, fieldname) {
							vb().Key().ClustCols().AddField(string(fieldname), datakind)
							ccComments(string(fieldname), comments)
						} else if contains(view.pkRef.PartitionKeyFields, fieldname) {
							vb().Key().Partition().AddField(string(fieldname), datakind)
							pkComments(string(fieldname), comments)
						} else {
							vb().Value().AddField(string(fieldname), datakind, f.Field.NotNull)
							valComments(string(fieldname), comments)
						}

					}
				} else if f.RefField != nil {
					fieldname := f.RefField.Name
					comments := f.RefField.Statement.GetComments()
					refs := make([]appdef.QName, 0)
					errors := false
					for i := range f.RefField.RefDocs {
						err := resolveInCtx(f.RefField.RefDocs[i], ictx, func(tbl *TableStmt, pkg *PackageSchemaAST) error {
							if e := c.checkReference(f.RefField.RefDocs[i], pkg, tbl, ictx); e != nil {
								return e
							}
							refs = append(refs, appdef.NewQName(string(pkg.Name), string(f.RefField.RefDocs[i].Name)))
							return nil
						})
						if err != nil {
							c.stmtErr(&f.RefField.Pos, err)
							errors = true
							continue
						}
					}
					if !errors {
						if contains(view.pkRef.ClusteringColumnsFields, fieldname) {
							vb().Key().ClustCols().AddRefField(string(fieldname), refs...)
							ccComments(string(fieldname), comments)
						} else if contains(view.pkRef.PartitionKeyFields, fieldname) {
							vb().Key().Partition().AddRefField(string(fieldname), refs...)
							pkComments(string(fieldname), comments)
						} else {
							vb().Value().AddRefField(string(fieldname), f.RefField.NotNull, refs...)
							valComments(string(fieldname), comments)
						}
					}
				}
			}
			c.popDef()
		})
	}
	return nil
}

func (c *buildContext) commands() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(cmd *CommandStmt, ictx *iterateCtx) {
			qname := schema.NewQName(cmd.Name)
			b := c.builder.AddCommand(qname)
			c.addComments(cmd, b)
			if cmd.Arg != nil && cmd.Arg.Def != nil {
				argQname := buildQname(ictx, cmd.Arg.Def.Package, cmd.Arg.Def.Name)
				b.SetArg(argQname)
			}
			if cmd.UnloggedArg != nil && cmd.UnloggedArg.Def != nil {
				argQname := buildQname(ictx, cmd.UnloggedArg.Def.Package, cmd.UnloggedArg.Def.Name)
				b.SetUnloggedArg(argQname)
			}
			if cmd.Returns != nil && cmd.Returns.Def != nil {
				retQname := buildQname(ictx, cmd.Returns.Def.Package, cmd.Returns.Def.Name)
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
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(q *QueryStmt, ictx *iterateCtx) {
			qname := schema.NewQName(q.Name)
			b := c.builder.AddQuery(qname)
			c.addComments(q, b)
			if q.Arg != nil && q.Arg.Def != nil {
				argQname := buildQname(ictx, q.Arg.Def.Package, q.Arg.Def.Name)
				b.SetArg(argQname)
			}

			if q.Returns.Any {
				b.SetResult(istructs.QNameANY)
			} else {
				if q.Returns.Def != nil {
					retQname := buildQname(ictx, q.Returns.Def.Package, q.Returns.Def.Name)
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
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(table *TableStmt, ictx *iterateCtx) {
			c.table(schema, table, ictx)
		})
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			c.workspaceDescriptor(schema, w, ictx)
		})
	}
	return errors.Join(c.errs...)
}

func (c *buildContext) fillTable(table *TableStmt, ictx *iterateCtx) {
	if table.Inherits != nil {
		if err := resolveInCtx(*table.Inherits, ictx, func(t *TableStmt, schema *PackageSchemaAST) error {
			c.fillTable(t, ictx)
			return nil
		}); err != nil {
			c.stmtErr(&table.Pos, err)
		}
	}
	c.addTableItems(table.Items, ictx)
}

func (c *buildContext) workspaceDescriptor(schema *PackageSchemaAST, w *WorkspaceStmt, ictx *iterateCtx) {
	if w.Descriptor != nil {
		qname := ictx.pkg.NewQName(w.Descriptor.Name)
		if c.isExists(qname, appdef.TypeKind_CDoc) {
			return
		}
		c.pushDef(qname, appdef.TypeKind_CDoc)
		c.addComments(w.Descriptor, c.defCtx().defBuilder.(appdef.ICommentBuilder))
		c.addTableItems(w.Descriptor.Items, ictx)
		c.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
		c.popDef()
	}
}

func (c *buildContext) table(schema *PackageSchemaAST, table *TableStmt, ictx *iterateCtx) {
	if isPredefinedSysTable(ictx.pkg.QualifiedPackageName, table) {
		return
	}

	qname := schema.NewQName(table.Name)
	if c.isExists(qname, table.tableTypeKind) {
		return
	}
	c.pushDef(qname, table.tableTypeKind)
	c.addComments(table, c.defCtx().defBuilder.(appdef.ICommentBuilder))
	c.fillTable(table, ictx)
	if table.singletone {
		c.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
	}
	if table.Abstract {
		c.defCtx().defBuilder.(appdef.IWithAbstractBuilder).SetAbstract()
	}
	c.popDef()
}

func (c *buildContext) addFieldRefToDef(refField *RefFieldExpr, ictx *iterateCtx) {
	if err := c.defCtx().checkName(string(refField.Name)); err != nil {
		c.stmtErr(&refField.Pos, err)
		return
	}
	refs := make([]appdef.QName, 0)
	errors := false
	for i := range refField.RefDocs {
		err := resolveInCtx(refField.RefDocs[i], ictx, func(tbl *TableStmt, pkg *PackageSchemaAST) error {
			if e := c.checkReference(refField.RefDocs[i], pkg, tbl, ictx); e != nil {
				return e
			}
			refs = append(refs, appdef.NewQName(string(pkg.Name), string(refField.RefDocs[i].Name)))
			return nil
		})
		if err != nil {
			c.stmtErr(&refField.Pos, err)
			errors = true
			continue
		}
	}
	if !errors {
		c.defCtx().defBuilder.(appdef.IFieldsBuilder).AddRefField(string(refField.Name), refField.NotNull, refs...)
	}
}

func (c *buildContext) addFieldToDef(field *FieldExpr, ictx *iterateCtx) {

	if field.Type.DataType != nil { // embedded type
		if err := c.defCtx().checkName(string(field.Name)); err != nil {
			c.stmtErr(&field.Pos, err)
			return
		}

		bld := c.defCtx().defBuilder.(appdef.IFieldsBuilder)
		fieldName := string(field.Name)
		sysDataKind := dataTypeToDataKind(*field.Type.DataType)

		if field.Type.DataType.Bytes != nil {
			if field.Type.DataType.Bytes.MaxLen != nil {
				bld.AddBytesField(fieldName, field.NotNull, appdef.MaxLen(*field.Type.DataType.Bytes.MaxLen))
			} else {
				bld.AddBytesField(fieldName, field.NotNull)
			}
		} else if field.Type.DataType.Varchar != nil {
			restricts := make([]appdef.IFieldRestrict, 0)
			if field.Type.DataType.Varchar.MaxLen != nil {
				restricts = append(restricts, appdef.MaxLen(*field.Type.DataType.Varchar.MaxLen))
			}
			if field.CheckRegexp != nil {
				restricts = append(restricts, appdef.Pattern(*field.CheckRegexp))
			}
			bld.AddStringField(fieldName, field.NotNull, restricts...)
		} else {
			bld.AddField(fieldName, sysDataKind, field.NotNull)
		}

		if field.Verifiable {
			bld.SetFieldVerify(fieldName, appdef.VerificationKind_EMail)
			// TODO: Support different verification kindsbuilder, &c
		}

		comments := field.Statement.GetComments()
		if len(comments) > 0 {
			bld.SetFieldComment(fieldName, comments...)
		}

	} else { // field.Type.Def
		// Record?
		pkg := string(field.Type.Def.Package)
		if pkg == "" {
			pkg = ictx.pkg.Name
		}
		qname := appdef.NewQName(pkg, string(field.Type.Def.Name))
		wrec := c.builder.WRecord(qname)
		crec := c.builder.CRecord(qname)
		orec := c.builder.ORecord(qname)

		if wrec == nil && orec == nil && crec == nil { // not yet built
			tbl, _, err := lookupInCtx[*TableStmt](DefQName{Package: Ident(qname.Pkg()), Name: Ident(qname.Entity())}, ictx)
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
			if tbl.tableTypeKind == appdef.TypeKind_CRecord || tbl.tableTypeKind == appdef.TypeKind_ORecord || tbl.tableTypeKind == appdef.TypeKind_WRecord {
				c.table(ictx.pkg, tbl, ictx)
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
			if (wrec != nil && tk != appdef.TypeKind_WRecord) ||
				(orec != nil && tk != appdef.TypeKind_ORecord) ||
				(crec != nil && tk != appdef.TypeKind_CRecord) {
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
		f := c.defCtx().defBuilder.(appdef.IFields).Field(string(constraint.UniqueField.Field))
		if f == nil {
			c.stmtErr(&constraint.Pos, ErrUndefinedField(string(constraint.UniqueField.Field)))
			return
		}
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).SetUniqueField(string(constraint.UniqueField.Field))
	}
}

func (c *buildContext) addNestedTableToDef(nested *NestedTableStmt, ictx *iterateCtx) {
	nestedTable := &nested.Table
	if nestedTable.tableTypeKind == appdef.TypeKind_null {
		c.stmtErr(&nestedTable.Pos, ErrUndefinedTableKind)
		return
	}

	containerName := string(nested.Name)
	if err := c.defCtx().checkName(containerName); err != nil {
		c.stmtErr(&nested.Pos, err)
		return
	}

	contQName := ictx.pkg.NewQName(nestedTable.Name)
	if !c.isExists(contQName, nestedTable.tableTypeKind) {
		c.pushDef(contQName, nestedTable.tableTypeKind)
		c.addTableItems(nestedTable.Items, ictx)
		c.popDef()
	}

	c.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(containerName, contQName, 0, maxNestedTableContainerOccurrences)

}
func (c *buildContext) addTableItems(items []TableItemExpr, ictx *iterateCtx) {
	for _, item := range items {
		if item.RefField != nil {
			c.addFieldRefToDef(item.RefField, ictx)
		} else if item.Field != nil {
			c.addFieldToDef(item.Field, ictx)
		} else if item.Constraint != nil {
			c.addConstraintToDef(item.Constraint)
		} else if item.NestedTable != nil {
			c.addNestedTableToDef(item.NestedTable, ictx)
		} else if item.FieldSet != nil {
			c.addFieldsOf(&item.FieldSet.Pos, item.FieldSet.Type, ictx)
		}
	}
}

func (c *buildContext) addFieldsOf(pos *lexer.Position, of DefQName, ictx *iterateCtx) {
	if err := resolveInCtx(of, ictx, func(t *TypeStmt, schema *PackageSchemaAST) error {
		c.addTableItems(t.Items, ictx)
		return nil
	}); err != nil {
		c.stmtErr(pos, err)
	}
}

type defBuildContext struct {
	defBuilder interface{}
	qname      appdef.QName
	kind       appdef.TypeKind
	names      map[string]bool
}

func (c *defBuildContext) checkName(name string) error {
	if _, ok := c.names[name]; ok {
		return ErrRedefined(name)
	}
	c.names[name] = true
	return nil
}

func (c *buildContext) pushDef(qname appdef.QName, kind appdef.TypeKind) {
	var builder interface{}
	switch kind {
	case appdef.TypeKind_CDoc:
		builder = c.builder.AddCDoc(qname)
	case appdef.TypeKind_CRecord:
		builder = c.builder.AddCRecord(qname)
	case appdef.TypeKind_ODoc:
		builder = c.builder.AddODoc(qname)
	case appdef.TypeKind_ORecord:
		builder = c.builder.AddORecord(qname)
	case appdef.TypeKind_WDoc:
		builder = c.builder.AddWDoc(qname)
	case appdef.TypeKind_WRecord:
		builder = c.builder.AddWRecord(qname)
	case appdef.TypeKind_Object:
		builder = c.builder.AddObject(qname)
	case appdef.TypeKind_ViewRecord:
		builder = c.builder.AddView(qname)
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

func (c *buildContext) isExists(qname appdef.QName, kind appdef.TypeKind) (exists bool) {
	switch kind {
	case appdef.TypeKind_CDoc:
		return c.builder.CDoc(qname) != nil
	case appdef.TypeKind_CRecord:
		return c.builder.CRecord(qname) != nil
	case appdef.TypeKind_ODoc:
		return c.builder.ODoc(qname) != nil
	case appdef.TypeKind_ORecord:
		return c.builder.ORecord(qname) != nil
	case appdef.TypeKind_WDoc:
		return c.builder.WDoc(qname) != nil
	case appdef.TypeKind_WRecord:
		return c.builder.WRecord(qname) != nil
	case appdef.TypeKind_Object:
		return c.builder.Object(qname) != nil
	default:
		panic(fmt.Sprintf("unsupported def kind %d", kind))
	}
}

func (c *buildContext) fundSchemaByPkg(pkg string) *PackageSchemaAST {
	for _, ast := range c.app.Packages {
		if ast.Name == pkg {
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

func (c *buildContext) checkReference(refTable DefQName, pkg *PackageSchemaAST, table *TableStmt, ictx *iterateCtx) error {
	if refTable.Package == "" {
		refTable.Package = Ident(pkg.Name)
	}
	refTableType := c.builder.TypeByName(appdef.NewQName(string(refTable.Package), string(refTable.Name)))
	if refTableType == nil {
		c.table(c.fundSchemaByPkg(string(refTable.Package)), table, ictx)
		refTableType = c.builder.TypeByName(appdef.NewQName(string(refTable.Package), string(refTable.Name)))
	}

	if refTableType == nil {
		//if it happened it means that error occurred
		return nil
	}

	for _, k := range canNotReferenceTo[c.defCtx().kind] {
		if k == refTableType.Kind() {
			return fmt.Errorf("table %s can not reference to table %s", c.defCtx().qname, refTableType.QName())
		}
	}

	return nil
}
