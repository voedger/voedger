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
}

func newBuildContext(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder) *buildContext {
	return &buildContext{
		basicContext: basicContext{
			app:  appSchema,
			errs: make([]error, 0),
		},
		builder: builder,
		defs:    make([]defBuildContext, 0),
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
		c.jobs,
		c.roles,
		c.queries,
		c.workspaces,
		c.grantsAndRevokes,
		c.packages,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}

	}
	return errors.Join(c.errs...)
}

func supported(stmt interface{}) bool {
	// FIXME: this must be empty in the end
	if _, ok := stmt.(*TagStmt); ok {
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

type wsBuilder struct {
	w   *WorkspaceStmt
	bld appdef.IWorkspaceBuilder
	pkg *PackageSchemaAST
}

func (c *buildContext) workspaces() error {
	wsBuilders := make([]wsBuilder, 0)

	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			qname := schema.NewQName(w.Name)
			bld := c.builder.AddWorkspace(qname)
			wsBuilders = append(wsBuilders, wsBuilder{w, bld, schema})
		})
	}

	for i := range wsBuilders {
		wb := wsBuilders[i]
		c.addComments(wb.w, wb.bld)
		if wb.w.Abstract {
			wb.bld.SetAbstract()
		}
		if wb.w.Descriptor != nil {
			wb.bld.SetDescriptor(wb.pkg.NewQName(wb.w.Descriptor.Name))
		}
		for qn := range wb.w.nodes {
			wb.bld.AddType(qn)
		}
	}
	return nil
}

func (c *buildContext) addComments(s IStatement, builder appdef.ICommentsBuilder) {
	comments := s.GetComments()
	if len(comments) > 0 {
		builder.SetComment(comments...)
	}
}

func (c *buildContext) types() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(typ *TypeStmt, ictx *iterateCtx) {
			c.pushDef(schema.NewQName(typ.Name), appdef.TypeKind_Object)
			c.addComments(typ, c.defCtx().defBuilder.(appdef.ICommentsBuilder))
			c.addTableItems(typ.Items, ictx)
			c.popDef()
		})
	}
	return nil
}

func (c *buildContext) roles() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(role *RoleStmt, ictx *iterateCtx) {
			rb := c.builder.AddRole(schema.NewQName(role.Name))
			c.addComments(role, rb)
		})
	}
	return nil

}

func (c *buildContext) grantsAndRevokes() error {
	grants := func(stmts []WorkspaceStatement) {
		for _, s := range stmts {
			if s.Grant != nil && len(s.Grant.on) > 0 {
				comments := s.Grant.GetComments()
				if (s.Grant.AllTablesWithTag != nil && s.Grant.AllTablesWithTag.All) ||
					(s.Grant.Table != nil && s.Grant.Table.All != nil) ||
					(s.Grant.AllTables != nil && s.Grant.AllTables.All) {
					c.builder.GrantAll(s.Grant.on, s.Grant.toRole, comments...)
					continue
				}
				c.builder.Grant(s.Grant.ops, s.Grant.on, s.Grant.columns, s.Grant.toRole, comments...)
			}
		}
	}
	revokes := func(stmts []WorkspaceStatement) {
		for _, s := range stmts {
			if s.Revoke != nil && len(s.Revoke.on) > 0 {
				comments := s.Revoke.GetComments()
				if (s.Revoke.AllTablesWithTag != nil && s.Revoke.AllTablesWithTag.All) ||
					(s.Revoke.Table != nil && s.Revoke.Table.All != nil) ||
					(s.Revoke.AllTables != nil && s.Revoke.AllTables.All) {
					c.builder.RevokeAll(s.Revoke.on, s.Revoke.toRole, comments...)
					continue
				}
				c.builder.Revoke(s.Revoke.ops, s.Revoke.on, s.Revoke.columns, s.Revoke.toRole, comments...)
			}
		}
	}
	handleWorkspace := func(stmts []WorkspaceStatement) {
		grants(stmts)
		revokes(stmts)
	}

	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			for _, inheritedWs := range w.inheritedWorkspaces {
				handleWorkspace(inheritedWs.Statements)
			}
			handleWorkspace(w.Statements)
		})
	}
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(w *AlterWorkspaceStmt, ictx *iterateCtx) {
			handleWorkspace(w.Statements)
		})
	}
	return nil
}

func (c *buildContext) jobs() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(job *JobStmt, ictx *iterateCtx) {
			jQname := schema.NewQName(job.Name)
			builder := c.builder.AddJob(jQname)
			builder.SetCronSchedule(*job.CronSchedule)

			for _, state := range job.State {
				builder.States().Add(state.storageQName, state.entityQNames...)
			}

			for _, intent := range job.Intents {
				builder.Intents().Add(intent.storageQName, intent.entityQNames...)
			}

			c.addComments(job, builder)
			builder.SetName(job.GetName())
			if job.Engine.WASM {
				builder.SetEngine(appdef.ExtensionEngineKind_WASM)
			} else {
				builder.SetEngine(appdef.ExtensionEngineKind_BuiltIn)
			}
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
					builder.Events().Add(qn, evKinds...)
				}
			}

			if proj.IncludingErrors {
				builder.SetWantErrors()
			}
			for _, intent := range proj.Intents {
				builder.Intents().Add(intent.storageQName, intent.entityQNames...)
			}
			for _, state := range proj.State {
				builder.States().Add(state.storageQName, state.entityQNames...)
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
						cc = append(cc, appdef.MaxLen(uint16(*f.Type.Bytes.MaxLen))) // nolint G115: checked in [analyseFields]
					}
				case appdef.DataKind_string:
					if (f.Type.Varchar != nil) && (f.Type.Varchar.MaxLen != nil) {
						cc = append(cc, appdef.MaxLen(uint16(*f.Type.Varchar.MaxLen))) // nolint G115: checked in [analyseFields]
					}
				}
				return cc
			}

			view.PartitionFields(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().Key().PartKey().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					vb().Key().PartKey().AddField(string(f.Field.Name.Value), dataTypeToDataKind(f.Field.Type))
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RefField != nil {
					vb().Key().PartKey().AddRefField(string(f.RefField.Name.Value), f.RefField.refQNames...)
					comment(f.RefField.Name.Value, f.RefField.Statement)
				}
			})

			view.ClusteringColumns(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().Key().ClustCols().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					k := dataTypeToDataKind(f.Field.Type)
					vb().Key().ClustCols().AddDataField(string(f.Field.Name.Value), appdef.SysDataName(k), resolveConstraints(f.Field)...)
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RefField != nil {
					vb().Key().ClustCols().AddRefField(string(f.RefField.Name.Value), f.RefField.refQNames...)
					comment(f.RefField.Name.Value, f.RefField.Statement)
				}
			})

			view.ValueFields(func(f *ViewItemExpr) {
				comment := func(n Ident, s Statement) {
					if txt := s.GetComments(); len(txt) > 0 {
						vb().Value().SetFieldComment(string(n), txt...)
					}
				}
				if f.Field != nil {
					k := dataTypeToDataKind(f.Field.Type)
					vb().Value().AddDataField(string(f.Field.Name.Value), appdef.SysDataName(k), f.Field.NotNull, resolveConstraints(f.Field)...)
					comment(f.Field.Name.Value, f.Field.Statement)
					return
				}
				if f.RecordField != nil {
					vb().Value().AddDataField(string(f.RecordField.Name.Value), appdef.SysDataName(appdef.DataKind_Record), f.RecordField.NotNull, []appdef.IConstraint{}...)
					comment(f.RecordField.Name.Value, f.RecordField.Statement)
				}
				if f.RefField != nil {
					vb().Value().AddRefField(string(f.RefField.Name.Value), f.RefField.NotNull, f.RefField.refQNames...)
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
			for _, intent := range cmd.Intents {
				b.Intents().Add(intent.storageQName, intent.entityQNames...)
			}
			for _, state := range cmd.State {
				b.States().Add(state.storageQName, state.entityQNames...)
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

			for _, state := range q.State {
				b.States().Add(state.storageQName, state.entityQNames...)
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
		c.addComments(w.Descriptor, c.defCtx().defBuilder.(appdef.ICommentsBuilder))
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
	c.addComments(table, c.defCtx().defBuilder.(appdef.ICommentsBuilder))
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
	fieldName := appdef.FieldName(field.Name)
	sysDataKind := dataTypeToDataKind(*field.Type.DataType)

	if field.Type.DataType.Bytes != nil {
		if field.Type.DataType.Bytes.MaxLen != nil {
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull, appdef.MaxLen(uint16(*field.Type.DataType.Bytes.MaxLen))) // nolint G115: checked in [analyseFields]
		} else {
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull)
		}
	} else if field.Type.DataType.Varchar != nil {
		constraints := make([]appdef.IConstraint, 0)
		if field.Type.DataType.Varchar.MaxLen != nil {
			constraints = append(constraints, appdef.MaxLen(uint16(*field.Type.DataType.Varchar.MaxLen))) // nolint G115: checked in [analyseFields]
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

	minOccur := appdef.Occurs(0)
	if field.NotNull {
		minOccur = 1
	}

	maxOccur := appdef.Occurs(1)
	// not supported by kernel yet
	// if field.Type.Array != nil {
	// 	if field.Type.Array.Unbounded {
	// 		maxOccur = maxNestedTableContainerOccurrences
	// 	} else {
	// 		maxOccur = field.Type.Array.MaxOccurs
	// 	}
	// }
	c.defCtx().defBuilder.(appdef.IObjectBuilder).AddContainer(string(field.Name), field.Type.qName, minOccur, maxOccur)
}

func (c *buildContext) addTableFieldToTable(field *FieldExpr, ictx *iterateCtx) {
	// Record?

	appDef := c.builder.AppDef()

	wrec := appDef.WRecord(field.Type.qName)
	crec := appDef.CRecord(field.Type.qName)
	orec := appDef.ORecord(field.Type.qName)

	if wrec == nil && orec == nil && crec == nil { // not yet built
		c.table(field.Type.tablePkg, field.Type.tableStmt, ictx)
		wrec = appDef.WRecord(field.Type.qName)
		crec = appDef.CRecord(field.Type.qName)
		orec = appDef.ORecord(field.Type.qName)
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
	tabName := c.defCtx().qname
	tab := c.builder.AppDef().Type(tabName)
	if constraint.UniqueField != nil {
		f := tab.(appdef.IFields).Field(string(constraint.UniqueField.Field))
		if f == nil {
			c.stmtErr(&constraint.Pos, ErrUndefinedField(string(constraint.UniqueField.Field)))
			return
		}
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).SetUniqueField(string(constraint.UniqueField.Field))
	} else if constraint.Unique != nil {
		fields := make([]string, len(constraint.Unique.Fields))
		for i, f := range constraint.Unique.Fields {
			if tab.(appdef.IFields).Field(string(f)) == nil {
				c.stmtErr(&constraint.Pos, ErrUndefinedField(string(f)))
				return
			}
			fields[i] = string(f)
		}
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
	appDef := c.builder.AppDef()
	switch kind {
	case appdef.TypeKind_CDoc:
		return appDef.CDoc(qname) != nil
	case appdef.TypeKind_CRecord:
		return appDef.CRecord(qname) != nil
	case appdef.TypeKind_ODoc:
		return appDef.ODoc(qname) != nil
	case appdef.TypeKind_ORecord:
		return appDef.ORecord(qname) != nil
	case appdef.TypeKind_WDoc:
		return appDef.WDoc(qname) != nil
	case appdef.TypeKind_WRecord:
		return appDef.WRecord(qname) != nil
	case appdef.TypeKind_Object:
		return appDef.Object(qname) != nil
	default:
		panic(fmt.Sprintf("unsupported type kind %d of %s", kind, qname))
	}
}

func (c *buildContext) popDef() {
	c.defs = c.defs[:len(c.defs)-1]
}

func (c *buildContext) defCtx() *defBuildContext {
	return &c.defs[len(c.defs)-1]
}

func (c *buildContext) checkReference(pkg *PackageSchemaAST, table *TableStmt) error {
	appDef := c.builder.AppDef()

	refTableType := appDef.TypeByName(appdef.NewQName(pkg.Name, string(table.Name)))
	if refTableType == nil {
		tableCtx := &iterateCtx{
			basicContext: &c.basicContext,
			collection:   pkg.Ast,
			pkg:          pkg,
			parent:       nil,
		}

		c.table(pkg, table, tableCtx)
		refTableType = appDef.TypeByName(appdef.NewQName(pkg.Name, string(table.Name)))
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
