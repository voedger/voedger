/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"errors"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/istructs"
)

type buildContext struct {
	basicContext
	adb  appdef.IAppDefBuilder
	defs []defBuildContext
}

func newBuildContext(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder) *buildContext {
	return &buildContext{
		basicContext: basicContext{
			app:  appSchema,
			errs: make([]error, 0),
		},
		adb:  builder,
		defs: make([]defBuildContext, 0),
	}
}

type buildFunc func() error

func (c *buildContext) build() error {
	c.prepareWSBuilders()

	var steps = []buildFunc{
		c.tags,
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
		c.limits,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return errors.Join(c.errs...)
}

// Prepares workspaces builders.
// First should be called during build stage, then w.builder should be used in next steps.
func (c *buildContext) prepareWSBuilders() {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			w.qName = schema.NewQName(w.Name)
			switch w.qName {
			case appdef.SysWorkspaceQName:
				w.builder = c.adb.AlterWorkspace(w.qName)
			default:
				w.builder = c.adb.AddWorkspace(w.qName)
			}
		})
	}
}

func (c *buildContext) packages() error {
	for localName, fullPath := range c.app.LocalNameToFullPath {
		c.adb.AddPackage(localName, fullPath)
	}
	return nil
}

func (c *buildContext) rates() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(rate *RateStmt, ictx *iterateCtx) {
			var timeUnitAmount uint32 = 1
			var period time.Duration
			var rateScopes []appdef.RateScope

			if rate.Value.TimeUnitAmounts != nil {
				timeUnitAmount = *rate.Value.TimeUnitAmounts
			}
			if rate.Value.TimeUnit.Second {
				period = time.Duration(timeUnitAmount) * time.Second
			} else if rate.Value.TimeUnit.Minute {
				period = time.Duration(timeUnitAmount) * time.Minute
			} else if rate.Value.TimeUnit.Hour {
				period = time.Duration(timeUnitAmount) * time.Hour
			} else if rate.Value.TimeUnit.Day {
				period = time.Duration(timeUnitAmount) * 24 * time.Hour
			} else if rate.Value.TimeUnit.Year {
				period = time.Duration(timeUnitAmount) * 365 * 24 * time.Hour
			}
			wsb := rate.workspace.mustBuilder(c)
			if rate.ObjectScope != nil {
				if rate.ObjectScope.PerAppPartition {
					rateScopes = append(rateScopes, appdef.RateScope_AppPartition)
				} else {
					rateScopes = append(rateScopes, appdef.RateScope_Workspace)
				}
			} else {
				rateScopes = append(rateScopes, appdef.RateScope_AppPartition) // default
			}
			if rate.SubjectScope != nil {
				if rate.SubjectScope.PerSubject {
					rateScopes = append(rateScopes, appdef.RateScope_User)
				} else {
					rateScopes = append(rateScopes, appdef.RateScope_IP)
				}
			}
			wsb.AddRate(schema.NewQName(rate.Name), rate.Value.count, period, rateScopes, rate.Comments...)
		})
	}
	return nil
}

func (c *buildContext) limits() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(limit *LimitStmt, ictx *iterateCtx) {
			wsb := limit.workspace.mustBuilder(c)
			var opt appdef.LimitFilterOption
			var types []appdef.TypeKind
			var limitFilter appdef.IFilter
			if limit.AllItems != nil {
				opt = appdef.LimitFilterOption_ALL
				if limit.AllItems.Commands {
					types = append(types, appdef.TypeKind_Command)
				}
				if limit.AllItems.Queries {
					types = append(types, appdef.TypeKind_Query)
				}
				if limit.AllItems.Tables {
					types = append(types, appdef.TypeKind_Records.AsArray()...)
				}
				if limit.AllItems.Views {
					types = append(types, appdef.TypeKind_ViewRecord)
				}
				if limit.AllItems.WithTag != nil {
					limitFilter = filter.And(filter.Types(types...), filter.Tags(limit.AllItems.WithTag.qName))
				} else {
					limitFilter = filter.Types(types...)
				}
			} else if limit.EachItem != nil {
				opt = appdef.LimitFilterOption_EACH
				if limit.EachItem.Commands {
					types = append(types, appdef.TypeKind_Command)
				}
				if limit.EachItem.Queries {
					types = append(types, appdef.TypeKind_Query)
				}
				if limit.EachItem.Tables {
					types = append(types, appdef.TypeKind_Records.AsArray()...)
				}
				if limit.EachItem.Views {
					types = append(types, appdef.TypeKind_ViewRecord)
				}
				if limit.EachItem.WithTag != nil {
					limitFilter = filter.And(filter.Types(types...), filter.Tags(limit.EachItem.WithTag.qName))
				} else {
					limitFilter = filter.Types(types...)
				}
			} else {
				opt = appdef.LimitFilterOption_EACH
				if limit.SingleItem.Command != nil {
					limitFilter = filter.And(filter.Types(appdef.TypeKind_Command), filter.QNames(limit.SingleItem.Command.qName))
				} else if limit.SingleItem.Query != nil {
					limitFilter = filter.And(filter.Types(appdef.TypeKind_Query), filter.QNames(limit.SingleItem.Query.qName))
				} else if limit.SingleItem.Table != nil {
					limitFilter = filter.And(filter.Types(appdef.TypeKind_Records.AsArray()...), filter.QNames(limit.SingleItem.Table.qName))
				} else if limit.SingleItem.View != nil {
					limitFilter = filter.And(filter.Types(appdef.TypeKind_ViewRecord), filter.QNames(limit.SingleItem.View.qName))
				}
			}
			wsb.AddLimit(schema.NewQName(limit.Name), limit.ops, opt, limitFilter, limit.RateName.qName, limit.Comments...)
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
			wsBuilders = append(wsBuilders, wsBuilder{w, w.builder, schema})
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

		ancestors := make([]appdef.QName, 0)
		for _, ancWS := range wb.w.inheritedWorkspaces {
			ancestors = append(ancestors, ancWS.qName)
		}
		if len(ancestors) > 0 {
			wb.bld.SetAncestors(ancestors[0], ancestors[1:]...)
		}

		for _, usedWS := range wb.w.usedWorkspaces {
			wb.bld.UseWorkspace(usedWS.qName)
		}

		// for qn := range wb.w.nodes {
		// 	wb.bld.AddType(qn)
		// }
	}
	return nil
}

func (c *buildContext) addComments(s IStatement, builder appdef.ICommenter) {
	comments := s.GetComments()
	if len(comments) > 0 {
		builder.SetComment(comments...)
	}
}

func (c *buildContext) tags() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(tag *TagStmt, ictx *iterateCtx) {
			qname := schema.NewQName(tag.Name)
			builder := tag.workspace.mustBuilder(c)
			featureAndComments := make([]string, 0)
			featureAndComments = append(featureAndComments, tag.Feature)
			featureAndComments = append(featureAndComments, tag.Comments...)
			builder.AddTag(qname, featureAndComments...)
		})
	}
	return nil
}

func (c *buildContext) types() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(typ *TypeStmt, ictx *iterateCtx) {
			c.pushDef(schema.NewQName(typ.Name), appdef.TypeKind_Object, typ.workspace)
			c.addComments(typ, c.defCtx().defBuilder.(appdef.ICommenter))
			c.addTableItems(schema, typ.Items)
			c.popDef()
		})
	}
	return nil
}

func (c *buildContext) roles() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(role *RoleStmt, ictx *iterateCtx) {
			wsb := role.workspace.mustBuilder(c)
			rb := wsb.AddRole(schema.NewQName(role.Name))
			if role.Published {
				rb.SetPublished(true)
			}
			c.addComments(role, rb)
		})
	}
	return nil

}

func (c *buildContext) grantsAndRevokes() error {
	grants := func(stmts []WorkspaceStatement) {
		for _, s := range stmts {
			if s.Grant != nil {
				wsb := s.Grant.workspace.mustBuilder(c)
				comments := s.Grant.GetComments()
				if (s.Grant.AllTablesWithTag != nil && s.Grant.AllTablesWithTag.All) ||
					(s.Grant.Table != nil && s.Grant.Table.All != nil) ||
					(s.Grant.AllTables != nil && s.Grant.AllTables.All) {
					wsb.Grant(grantAllToTableOps, s.Grant.filter(), []appdef.FieldName{}, s.Grant.toRole, comments...)
					continue
				}
				wsb.Grant(s.Grant.ops, s.Grant.filter(), s.Grant.columns, s.Grant.toRole, comments...)
			}
		}
	}
	revokes := func(stmts []WorkspaceStatement) {
		for _, s := range stmts {
			if s.Revoke != nil {
				wsb := s.Revoke.workspace.mustBuilder(c)
				comments := s.Revoke.GetComments()
				if (s.Revoke.AllTablesWithTag != nil && s.Revoke.AllTablesWithTag.All) ||
					(s.Revoke.Table != nil && s.Revoke.Table.All != nil) ||
					(s.Revoke.AllTables != nil && s.Revoke.AllTables.All) {
					wsb.Revoke(grantAllToTableOps, s.Revoke.filter(), []appdef.FieldName{}, s.Revoke.toRole, comments...)
					continue
				}
				wsb.Revoke(s.Revoke.ops, s.Revoke.filter(), s.Revoke.columns, s.Revoke.toRole, comments...)
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

			wsb := job.workspace.mustBuilder(c)
			builder := wsb.AddJob(jQname)
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

			wsb := proj.workspace.mustBuilder(c)
			builder := wsb.AddProjector(pQname)
			// Triggers
			for _, trigger := range proj.Triggers {
				ops := make([]appdef.OperationKind, 0)
				if trigger.ExecuteAction != nil {
					if trigger.ExecuteAction.WithParam {
						ops = append(ops, appdef.OperationKind_ExecuteWithParam)
					} else {
						ops = append(ops, appdef.OperationKind_Execute)
					}
				} else {
					if trigger.insert() {
						ops = append(ops, appdef.OperationKind_Insert)
					}
					if trigger.update() {
						ops = append(ops, appdef.OperationKind_Update)
					}
					if trigger.activate() {
						ops = append(ops, appdef.OperationKind_Activate)
					}
					if trigger.deactivate() {
						ops = append(ops, appdef.OperationKind_Deactivate)
					}
				}
				if len(ops) == 0 {
					c.errs = append(c.errs, fmt.Errorf("no trigger operations specified for projector %s", proj.Name))
					return
				}

				flt := []appdef.IFilter{}
				qNames := appdef.QNames{}
				types := []appdef.TypeKind{}
				for _, n := range trigger.QNames {
					switch n.qName {
					case istructs.QNameCRecord:
						types = append(types, appdef.TypeKind_CDoc, appdef.TypeKind_CRecord)
					case istructs.QNameWRecord:
						types = append(types, appdef.TypeKind_WDoc, appdef.TypeKind_WRecord)
					case istructs.QNameODoc:
						types = append(types, appdef.TypeKind_ODoc)
					default:
						qNames.Add(n.qName)
					}
				} //Trigger qNames

				if len(qNames) > 0 {
					flt = append(flt, filter.QNames(qNames...))
				}
				if len(types) > 0 {
					flt = append(flt, filter.Types(types...))
				}

				switch len(flt) {
				case 0:
					c.errs = append(c.errs, fmt.Errorf("no triggers names specified for projector %s", proj.Name))
					return
				case 1:
					builder.Events().Add(ops, flt[0])
				default:
					builder.Events().Add(ops, filter.Or(flt...))
				}
			} //Triggers

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
	return errors.Join(c.errs...)
}

func (c *buildContext) views() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(view *ViewStmt, ictx *iterateCtx) {
			c.pushDef(schema.NewQName(view.Name), appdef.TypeKind_ViewRecord, view.workspace)
			vb := func() appdef.IViewBuilder {
				return c.defCtx().defBuilder.(appdef.IViewBuilder)
			}
			c.addComments(view, vb())
			c.applyTags(view.With, c.defCtx().defBuilder.(appdef.ITagger))

			resolveConstraints := func(f *ViewField) []appdef.IConstraint {
				cc := []appdef.IConstraint{}
				switch k := dataTypeToDataKind(f.Type); k {
				case appdef.DataKind_bytes:
					if (f.Type.Bytes != nil) && (f.Type.Bytes.MaxLen != nil) {
						cc = append(cc, constraints.MaxLen(uint16(*f.Type.Bytes.MaxLen))) // nolint G115: checked in [analyseFields]
					}
				case appdef.DataKind_string:
					if (f.Type.Varchar != nil) && (f.Type.Varchar.MaxLen != nil) {
						cc = append(cc, constraints.MaxLen(uint16(*f.Type.Varchar.MaxLen))) // nolint G115: checked in [analyseFields]
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

			wsb := cmd.workspace.mustBuilder(c)
			b := wsb.AddCommand(qname)

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
			c.applyTags(cmd.With, b)
		})
	}
	return nil
}

func (c *buildContext) applyTags(with []WithItem, t appdef.ITagger) {
	for _, item := range with {
		if len(item.tags) > 0 {
			t.SetTag(item.tags...)
		}
	}
}

func (c *buildContext) queries() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(q *QueryStmt, ictx *iterateCtx) {
			qname := schema.NewQName(q.Name)

			wsb := q.workspace.mustBuilder(c)
			b := wsb.AddQuery(qname)

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
			c.applyTags(q.With, b)
		})
	}
	return nil
}

func (c *buildContext) tables() error {
	for _, schema := range c.app.Packages {
		iteratePackageStmt(schema, &c.basicContext, func(table *TableStmt, ictx *iterateCtx) {
			c.table(schema, table)
		})
		iteratePackageStmt(schema, &c.basicContext, func(w *WorkspaceStmt, ictx *iterateCtx) {
			c.workspaceDescriptor(w, ictx)
		})
	}
	return errors.Join(c.errs...)
}

func (c *buildContext) fillTable(schema *PackageSchemaAST, table *TableStmt) {
	if table.inherits.table != nil {
		c.fillTable(schema, table.inherits.table)
	}
	c.addTableItems(schema, table.Items)
}

func (c *buildContext) workspaceDescriptor(w *WorkspaceStmt, ictx *iterateCtx) {
	if w.Descriptor != nil {
		qname := ictx.pkg.NewQName(w.Descriptor.Name)
		if c.isExists(qname, appdef.TypeKind_CDoc) {
			return
		}
		c.pushDef(qname, appdef.TypeKind_CDoc, w.Descriptor.workspace)
		c.addComments(w.Descriptor, c.defCtx().defBuilder.(appdef.ICommenter))
		c.addTableItems(ictx.pkg, w.Descriptor.Items)
		c.defCtx().defBuilder.(appdef.ICDocBuilder).SetSingleton()
		c.popDef()
	}
}

func (c *buildContext) table(schema *PackageSchemaAST, table *TableStmt) {
	qname := schema.NewQName(table.Name)
	if c.isExists(qname, table.tableTypeKind) {
		return
	}
	c.pushDef(qname, table.tableTypeKind, table.workspace)
	c.addComments(table, c.defCtx().defBuilder.(appdef.ICommenter))
	c.fillTable(schema, table)
	if table.singleton {
		c.defCtx().defBuilder.(appdef.ISingletonBuilder).SetSingleton()
	}
	if table.Abstract {
		c.defCtx().defBuilder.(appdef.IWithAbstractBuilder).SetAbstract()
	}
	c.applyTags(table.With, c.defCtx().defBuilder.(appdef.ITagger))
	c.popDef()
}

func (c *buildContext) addFieldRefToDef(refField *RefFieldExpr) {
	if err := c.defCtx().checkName(string(refField.Name)); err != nil {
		c.stmtErr(&refField.Pos, err)
		return
	}
	for _, refTable := range refField.refTables {
		if err := c.checkReference(refTable.pkg, refTable.table); err != nil {
			c.stmtErr(&refField.Pos, err)
			return
		}
	}
	c.defCtx().defBuilder.(appdef.IFieldsBuilder).AddRefField(string(refField.Name), refField.NotNull, refField.refQNames...)
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
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull, constraints.MaxLen(uint16(*field.Type.DataType.Bytes.MaxLen))) // nolint G115: checked in [analyseFields]
		} else {
			bld.AddField(fieldName, appdef.DataKind_bytes, field.NotNull)
		}
	} else if field.Type.DataType.Varchar != nil {
		cc := make([]appdef.IConstraint, 0)
		if field.Type.DataType.Varchar.MaxLen != nil {
			cc = append(cc, constraints.MaxLen(uint16(*field.Type.DataType.Varchar.MaxLen))) // nolint G115: checked in [analyseFields]
		}
		if field.CheckRegexp != nil {
			cc = append(cc, constraints.Pattern(field.CheckRegexp.Regexp))
		}
		bld.AddField(fieldName, appdef.DataKind_string, field.NotNull, cc...)
	} else {
		bld.AddField(fieldName, sysDataKind, field.NotNull)
	}

	if field.Verifiable {
		bld.SetFieldVerify(fieldName, appdef.VerificationKind_EMail)
		// TODO: Support different verification kindsbuilder, &c
	}

	comments := field.GetComments()
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

func (c *buildContext) addTableFieldToTable(field *FieldExpr) {
	// Record?

	appDef := c.adb.AppDef()

	wrec := appdef.WRecord(appDef.Type, field.Type.qName)
	crec := appdef.CRecord(appDef.Type, field.Type.qName)
	orec := appdef.ORecord(appDef.Type, field.Type.qName)

	if wrec == nil && orec == nil && crec == nil { // not yet built
		c.table(field.Type.tablePkg, field.Type.tableStmt)
		wrec = appdef.WRecord(appDef.Type, field.Type.qName)
		crec = appdef.CRecord(appDef.Type, field.Type.qName)
		orec = appdef.ORecord(appDef.Type, field.Type.qName)
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

func (c *buildContext) addFieldToDef(field *FieldExpr) {
	if field.Type.DataType != nil {
		c.addDataTypeField(field)
	} else {
		if c.defCtx().kind == appdef.TypeKind_Object {
			c.addObjectFieldToType(field)
		} else {
			c.addTableFieldToTable(field)
		}
	}
}

func (c *buildContext) addConstraintToDef(constraint *TableConstraint) {
	tabName := c.defCtx().qname
	tab := c.adb.AppDef().Type(tabName)
	if constraint.UniqueField != nil {
		f := tab.(appdef.IWithFields).Field(string(constraint.UniqueField.Field))
		if f == nil {
			c.stmtErr(&constraint.Pos, ErrUndefinedField(string(constraint.UniqueField.Field)))
			return
		}
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).SetUniqueField(string(constraint.UniqueField.Field))
	} else if constraint.Unique != nil {
		fields := make([]string, len(constraint.Unique.Fields))
		for i, f := range constraint.Unique.Fields {
			if tab.(appdef.IWithFields).Field(string(f)) == nil {
				c.stmtErr(&constraint.Pos, ErrUndefinedField(string(f)))
				return
			}
			fields[i] = string(f)
		}
		c.defCtx().defBuilder.(appdef.IUniquesBuilder).AddUnique(appdef.UniqueQName(tabName, string(constraint.ConstraintName)), fields)
	}
}

func (c *buildContext) addNestedTableToDef(schema *PackageSchemaAST, nested *NestedTableStmt) {
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

	contQName := schema.NewQName(nestedTable.Name)
	if !c.isExists(contQName, nestedTable.tableTypeKind) {
		c.pushDef(contQName, nestedTable.tableTypeKind, nestedTable.workspace)
		c.addTableItems(schema, nestedTable.Items)
		c.applyTags(nestedTable.With, c.defCtx().defBuilder.(appdef.ITagger))
		c.popDef()
	}

	c.defCtx().defBuilder.(appdef.IContainersBuilder).AddContainer(containerName, contQName, 0, maxNestedTableContainerOccurrences)

}
func (c *buildContext) addTableItems(schema *PackageSchemaAST, items []TableItemExpr) {

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
			c.addFieldRefToDef(item.RefField)
		} else if item.Field != nil {
			c.addFieldToDef(item.Field)
		} else if item.Constraint != nil {
			c.addConstraintToDef(item.Constraint)
		} else if item.NestedTable != nil {
			c.addNestedTableToDef(schema, item.NestedTable)
		} else if item.FieldSet != nil {
			c.addTableItems(schema, item.FieldSet.typ.Items)
		}
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

func (c *buildContext) pushDef(qname appdef.QName, kind appdef.TypeKind, currentWorkspace workspaceAddr) {

	wsb := currentWorkspace.mustBuilder(c)

	var builder interface{}
	switch kind {
	case appdef.TypeKind_CDoc:
		builder = wsb.AddCDoc(qname)
	case appdef.TypeKind_CRecord:
		builder = wsb.AddCRecord(qname)
	case appdef.TypeKind_ODoc:
		builder = wsb.AddODoc(qname)
	case appdef.TypeKind_ORecord:
		builder = wsb.AddORecord(qname)
	case appdef.TypeKind_WDoc:
		builder = wsb.AddWDoc(qname)
	case appdef.TypeKind_WRecord:
		builder = wsb.AddWRecord(qname)
	case appdef.TypeKind_Object:
		builder = wsb.AddObject(qname)
	case appdef.TypeKind_ViewRecord:
		builder = wsb.AddView(qname)
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
	case appdef.TypeKind_CDoc,
		appdef.TypeKind_CRecord,
		appdef.TypeKind_ODoc,
		appdef.TypeKind_ORecord,
		appdef.TypeKind_WDoc,
		appdef.TypeKind_WRecord,
		appdef.TypeKind_Object:
		return appdef.TypeByNameAndKind[appdef.IRecord](c.adb.AppDef().Type, qname, kind) != nil
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
	appDef := c.adb.AppDef()

	refTableType := appdef.Structure(appDef.Type, appdef.NewQName(pkg.Name, string(table.Name)))
	if refTableType == nil {
		c.table(pkg, table)
		refTableType = appdef.Structure(appDef.Type, appdef.NewQName(pkg.Name, string(table.Name)))
	}

	if refTableType == nil {
		// if it happened it means that error occurred
		return nil
	}

	for _, k := range canNotReferenceTo[c.defCtx().kind] {
		if k == refTableType.Kind() {
			return fmt.Errorf("table %s can not reference to %v", c.defCtx().qname, refTableType)
		}
	}

	return nil
}
