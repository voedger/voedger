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
)

type buildContext struct {
	basicContext
	builder          appdef.IAppDefBuilder
	defs             []defBuildContext
	variableResolver IVariableResolver
	wsBuildCtxs      map[*WorkspaceStmt]*wsBuildCtx
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
		c.rates,
		c.tables,
		c.views,
		c.commands,
		c.projectors,
		c.queries,
		c.workspaces,
		c.alterWorkspaces,
		c.inheritedWorkspaces,
		c.packages,
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
	if _, ok := stmt.(*LimitStmt); ok {
		return false
	}
	return true
}

func (c *buildContext) useStmtInWs(wsctx *wsBuildCtx, stmtPackage string, stmt interface{}) {
	if named, ok := stmt.(INamedStatement); ok {
		if supported(stmt) {
			wsctx.builder.AddType(appdef.NewQName(stmtPackage, named.GetName()))
		}
	}
	if useTable, ok := stmt.(*UseTableStmt); ok {
		for _, qn := range useTable.qNames {
			wsctx.builder.AddType(qn)
		}
	}
	if useWorkspace, ok := stmt.(*UseWorkspaceStmt); ok {
		wsctx.builder.AddType(useWorkspace.qName)
	}
}

func (c *buildContext) packages() error {
	for localName, fullPath := range c.app.LocalNameToFullPath {
		c.builder.AddPackage(localName, fullPath)
	}
	return nil
}

func (c *buildContext) rates() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(rate *RateStmt, ictx *iterateCtx) {
			if rate.Value.Variable != nil {
				if c.variableResolver != nil {
					c.variableResolver.AsInt32(rate.Value.variable)
					// TODO: use in appdef builder
				}
			}
		})
	}
	return nil
}

func (c *buildContext) workspaces() error {

	var iter func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection)

	iter = func(ws *WorkspaceStmt, wsctx *wsBuildCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			c.useStmtInWs(wsctx, wsctx.pkg.Name, stmt)
			if collection, ok := stmt.(IStatementCollection); ok {
				if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
					iter(ws, wsctx, collection)
				}
			}
			if t, ok := stmt.(*TableStmt); ok {
				for i := range t.Items {
					if t.Items[i].NestedTable != nil {
						c.useStmtInWs(wsctx, wsctx.pkg.Name, &t.Items[i].NestedTable.Table)
						iter(ws, wsctx, &t.Items[i].NestedTable.Table)
					}
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
			wsc.builder.SetDescriptor(appdef.NewQName(wsc.ictx.pkg.Name, w.Descriptor.GetName()))
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
					c.useStmtInWs(wsctx, pkgAst.Name, stmt)
					if collection, ok := stmt.(IStatementCollection); ok {
						if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
							iter(wsctx, collection)
						}
					}
					if t, ok := stmt.(*TableStmt); ok {
						for i := range t.Items {
							if t.Items[i].NestedTable != nil {
								c.useStmtInWs(wsctx, wsctx.pkg.Name, &t.Items[i].NestedTable.Table)
								iter(wsctx, &t.Items[i].NestedTable.Table)
							}
						}
					}
				})
			}
			iter(c.wsBuildCtxs[a.alteredWorkspace], a)
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

func (c *buildContext) projectors() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(proj *ProjectorStmt, ictx *iterateCtx) {
			pQname := schema.NewQName(proj.Name)
			builder := c.builder.AddProjector(pQname)
			// Triggers
			for _, trigger := range proj.Triggers {
				evKinds := make([]appdef.ProjectorEventKind, 0)
				if trigger.ExecuteAction != nil {
					if trigger.ExecuteAction.WithParam {
						evKinds = append(evKinds, appdef.ProjectorEventKind_ExecuteWithParam)
					} else {
						evKinds = append(evKinds, appdef.ProjectorEventKind_Execute)
					}
				} else {
					if trigger.insert() {
						evKinds = append(evKinds, appdef.ProjectorEventKind_Insert)
					}
					if trigger.update() {
						evKinds = append(evKinds, appdef.ProjectorEventKind_Update)
					}
					if trigger.activate() {
						evKinds = append(evKinds, appdef.ProjectorEventKind_Activate)
					}
					if trigger.deactivate() {
						evKinds = append(evKinds, appdef.ProjectorEventKind_Deactivate)
					}
				}
				for _, qn := range trigger.qNames {
					builder.AddEvent(qn, evKinds...)
				}
			}

			if proj.IncludingErrors {
				builder.SetWantErrors()
			}
			for _, intent := range proj.Intents {
				builder.AddIntent(intent.storageQName, intent.entityQNames...)
			}
			for _, state := range proj.State {
				builder.AddState(state.storageQName, state.entityQNames...)
			}

			c.addComments(proj, builder)
			builder.SetName(proj.GetName())
			if proj.Engine.WASM {
				builder.SetEngine(appdef.ExtensionEngineKind_WASM)
			} else {
				builder.SetEngine(appdef.ExtensionEngineKind_BuiltIn)
			}
			builder.SetSync(proj.Sync)
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

			resolveConstraints := func(f *ViewField) []appdef.IConstraint {
				cc := []appdef.IConstraint{}
				switch k := dataTypeToDataKind(f.Type); k {
				case appdef.DataKind_bytes:
					if (f.Type.Bytes != nil) && (f.Type.Bytes.MaxLen != nil) {
						cc = append(cc, appdef.MaxLen(uint16(*f.Type.Bytes.MaxLen)))
					}
				case appdef.DataKind_string:
					if (f.Type.Varchar != nil) && (f.Type.Varchar.MaxLen != nil) {
						cc = append(cc, appdef.MaxLen(uint16(*f.Type.Varchar.MaxLen)))
					}
				}
				return cc
			}

			view.PartitionFields(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().KeyBuilder().PartKeyBuilder().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					vb().KeyBuilder().PartKeyBuilder().AddField(string(f.Field.Name.Value), dataTypeToDataKind(f.Field.Type))
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RefField != nil {
					vb().KeyBuilder().PartKeyBuilder().AddRefField(string(f.RefField.Name.Value), f.RefField.refQNames...)
					comment(f.RefField.Name.Value, f.RefField.Statement)
				}
			})

			view.ClusteringColumns(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().KeyBuilder().ClustColsBuilder().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					k := dataTypeToDataKind(f.Field.Type)
					vb().KeyBuilder().ClustColsBuilder().AddDataField(string(f.Field.Name.Value), appdef.SysDataName(k), resolveConstraints(f.Field)...)
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RefField != nil {
					vb().KeyBuilder().ClustColsBuilder().AddRefField(string(f.RefField.Name.Value), f.RefField.refQNames...)
					comment(f.RefField.Name.Value, f.RefField.Statement)
				}
			})

			view.ValueFields(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().ValueBuilder().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					k := dataTypeToDataKind(f.Field.Type)
					vb().ValueBuilder().AddDataField(string(f.Field.Name.Value), appdef.SysDataName(k), f.Field.NotNull, resolveConstraints(f.Field)...)
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RefField != nil {
					vb().ValueBuilder().AddRefField(string(f.RefField.Name.Value), f.RefField.NotNull, f.RefField.refQNames...)
					comment(f.RefField.Name.Value, f.RefField.Statement)
				}
			})
			c.popDef()
		})
	}
	return nil
}

func setParam(ictx *iterateCtx, v *AnyOrVoidOrDef, cb func(qn appdef.QName)) {
	if v.Def != nil {
		argQname := buildQname(ictx, v.Def.Package, v.Def.Name)
		cb(argQname)
	} else if v.Any {
		cb(appdef.QNameANY)
	}
}

func (c *buildContext) commands() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(cmd *CommandStmt, ictx *iterateCtx) {
			qname := schema.NewQName(cmd.Name)
			b := c.builder.AddCommand(qname)
			c.addComments(cmd, b)
			if cmd.Param != nil {
				setParam(ictx, cmd.Param, func(qn appdef.QName) { b.SetParam(qn) })
			}
			if cmd.UnloggedParam != nil {
				setParam(ictx, cmd.UnloggedParam, func(qn appdef.QName) { b.SetUnloggedParam(qn) })
			}
			if cmd.Returns != nil {
				setParam(ictx, cmd.Returns, func(qn appdef.QName) { b.SetResult(qn) })
			}
			b.SetName(cmd.GetName())
			if cmd.Engine.WASM {
				b.SetEngine(appdef.ExtensionEngineKind_WASM)
			} else {
				b.SetEngine(appdef.ExtensionEngineKind_BuiltIn)
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
			if q.Param != nil {
				setParam(ictx, q.Param, func(qn appdef.QName) { b.SetParam(qn) })
			}

			setParam(ictx, &q.Returns, func(qn appdef.QName) { b.SetResult(qn) })

			b.SetName(q.GetName())
			if q.Engine.WASM {
				b.SetEngine(appdef.ExtensionEngineKind_WASM)
			} else {
				b.SetEngine(appdef.ExtensionEngineKind_BuiltIn)
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
			c.workspaceDescriptor(w, ictx)
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

func (c *buildContext) workspaceDescriptor(w *WorkspaceStmt, ictx *iterateCtx) {
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
	qname := schema.NewQName(table.Name)
	if c.isExists(qname, table.tableTypeKind) {
		return
	}
	c.pushDef(qname, table.tableTypeKind)
	c.addComments(table, c.defCtx().defBuilder.(appdef.ICommentBuilder))
	c.fillTable(table, ictx)
	if table.singleton {
		c.defCtx().defBuilder.(appdef.ISingletonBuilder).SetSingleton()
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
			if e := c.checkReference(pkg, tbl); e != nil {
				return e
			}
			refs = append(refs, appdef.NewQName(pkg.Name, string(refField.RefDocs[i].Name)))
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

func (c *buildContext) addDataTypeField(field *FieldExpr) {
	if err := c.defCtx().checkName(string(field.Name)); err != nil {
		c.stmtErr(&field.Pos, err)
		return
	}

	bld := c.defCtx().defBuilder.(appdef.IFieldsBuilder)
	fieldName := string(field.Name)
	sysDataKind := dataTypeToDataKind(*field.Type.DataType)

	if field.Type.DataType.Bytes != nil {
		if field.Type.DataType.Bytes.MaxLen != nil {
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull, appdef.MaxLen(uint16(*field.Type.DataType.Bytes.MaxLen)))
		} else {
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull)
		}
	} else if field.Type.DataType.Varchar != nil {
		constraints := make([]appdef.IConstraint, 0)
		if field.Type.DataType.Varchar.MaxLen != nil {
			constraints = append(constraints, appdef.MaxLen(uint16(*field.Type.DataType.Varchar.MaxLen)))
		}
		if field.CheckRegexp != nil {
			constraints = append(constraints, appdef.Pattern(field.CheckRegexp.Regexp))
		}
		bld.AddField(fieldName, appdef.DataKind_string, field.NotNull, constraints...)
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
}

func (c *buildContext) addObjectFieldToType(field *FieldExpr) {

	minOccur := 0
	if field.NotNull {
		minOccur = 1
	}

	maxOccur := 1
	// not supported by kernel yet
	// if field.Type.Array != nil {
	// 	if field.Type.Array.Unbounded {
	// 		maxOccur = maxNestedTableContainerOccurrences
	// 	} else {
	// 		maxOccur = field.Type.Array.MaxOccurs
	// 	}
	// }
	c.defCtx().defBuilder.(appdef.IObjectBuilder).AddContainer(string(field.Name), field.Type.qName, appdef.Occurs(minOccur), appdef.Occurs(maxOccur))
}

func (c *buildContext) addTableFieldToTable(field *FieldExpr, ictx *iterateCtx) {
	// Record?

	wrec := c.builder.WRecord(field.Type.qName)
	crec := c.builder.CRecord(field.Type.qName)
	orec := c.builder.ORecord(field.Type.qName)

	if wrec == nil && orec == nil && crec == nil { // not yet built
		c.table(field.Type.tablePkg, field.Type.tableStmt, ictx)
		wrec = c.builder.WRecord(field.Type.qName)
		crec = c.builder.CRecord(field.Type.qName)
		orec = c.builder.ORecord(field.Type.qName)
	}

	if wrec != nil || orec != nil || crec != nil {
		tk, err := getNestedTableKind(c.defCtx().kind)
		if err != nil {
			c.stmtErr(&field.Pos, err)
			return
		}
		if (wrec != nil && tk != appdef.TypeKind_WRecord) ||
			(orec != nil && tk != appdef.TypeKind_ORecord) ||
			(crec != nil && tk != appdef.TypeKind_CRecord) {
			c.errs = append(c.errs, ErrNestedTableIncorrectKind)
			return
		}
		c.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(string(field.Name), field.Type.qName, 0, maxNestedTableContainerOccurrences)
	} else {
		c.stmtErr(&field.Pos, ErrTypeNotSupported(field.Type.String()))
	}
}

func (c *buildContext) addFieldToDef(field *FieldExpr, ictx *iterateCtx) {
	if field.Type.DataType != nil {
		c.addDataTypeField(field)
	} else {
		if c.defCtx().kind == appdef.TypeKind_Object {
			c.addObjectFieldToType(field)
		} else {
			c.addTableFieldToTable(field, ictx)
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
	} else if constraint.Unique != nil {
		fields := make([]string, len(constraint.Unique.Fields))
		for i, f := range constraint.Unique.Fields {
			fields[i] = string(f)
		}
		tabName := c.defCtx().defBuilder.(appdef.IType).QName()
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).AddUnique(appdef.UniqueQName(tabName, string(constraint.ConstraintName)), fields)
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

	func() {
		// generate unique names if empty
		const nameFmt = "%02d"
		cnt := 0
		for _, item := range items {
			if (item.Constraint != nil) && (item.Constraint.Unique != nil) {
				if item.Constraint.ConstraintName == "" {
					cnt++
					item.Constraint.ConstraintName = Ident(fmt.Sprintf(nameFmt, cnt))
				}
			}
		}
	}()

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
		panic(fmt.Sprintf("unsupported def kind %d of %s", kind, qname))
	}
}

func (c *buildContext) popDef() {
	c.defs = c.defs[:len(c.defs)-1]
}

func (c *buildContext) defCtx() *defBuildContext {
	return &c.defs[len(c.defs)-1]
}

func (c *buildContext) checkReference(pkg *PackageSchemaAST, table *TableStmt) error {
	refTableType := c.builder.TypeByName(appdef.NewQName(pkg.Name, string(table.Name)))
	if refTableType == nil {
		tableCtx := &iterateCtx{
			basicContext: &c.basicContext,
			collection:   pkg.Ast,
			pkg:          pkg,
			parent:       nil,
		}

		c.table(pkg, table, tableCtx)
		refTableType = c.builder.TypeByName(appdef.NewQName(pkg.Name, string(table.Name)))
	}

	if refTableType == nil {
		// if it happened it means that error occurred
		return nil
	}

	for _, k := range canNotReferenceTo[c.defCtx().kind] {
		if k == refTableType.Kind() {
			return fmt.Errorf("table %s can not reference to table %s", c.defCtx().qname, refTableType.QName())
		}
	}

	return nil
}
