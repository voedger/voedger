/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/robfig/cron/v3"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/set"
	"github.com/voedger/voedger/pkg/istructs"
)

type iterateCtx struct {
	*basicContext
	pkg        *PackageSchemaAST
	collection IStatementCollection
	parent     *iterateCtx
	wsCtxs     map[*WorkspaceStmt]*wsCtx
}

// Returns the current workspace.
//
// # Panics:
//   - if there is no current workspace.
func (c *iterateCtx) mustCurrentWorkspace() workspaceAddr {
	if ws := getCurrentWorkspace(c); (ws.workspace != nil) && (ws.pkg != nil) {
		return ws
	}
	panic("no current workspace")
}

func (c *iterateCtx) setPkg(pkg *PackageSchemaAST) {
	c.pkg = pkg
	c.collection = pkg.Ast
	c.parent = nil
}

func FindApplication(p *PackageSchemaAST) (result *ApplicationStmt, err error) {
	for _, stmt := range p.Ast.Statements {
		if stmt.Application != nil {
			if result != nil {
				return nil, fmt.Errorf("%s: %w", stmt.Application.Pos.String(), ErrApplicationRedefined)
			}
			result = stmt.Application
		}
	}
	return result, nil
}

func preAnalyse(c *basicContext, packages []*PackageSchemaAST) {
	for _, p := range packages {
		iteratePackage(p, c, func(stmt interface{}, ictx *iterateCtx) {
			switch v := stmt.(type) {
			case *TableStmt:
				preAnalyseTable(v, ictx)
			case *AlterWorkspaceStmt:
				preAnalyseAlterWorkspace(v, ictx)
			}
		})
	}
}

func analyse(c *basicContext, packages []*PackageSchemaAST) {
	wsIncludeCtxs := make(map[*WorkspaceStmt]*wsCtx)
	ictx := &iterateCtx{
		basicContext: c,
		parent:       nil,
		wsCtxs:       wsIncludeCtxs,
	}
	// Pass 1
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			switch v := stmt.(type) {
			case *ImportStmt:
				analyzeImport(v, ictx)
			case *TagStmt:
				analyzeTag(v, ictx)
			case *CommandStmt:
				analyzeCommand(v, ictx)
			case *QueryStmt:
				analyzeQuery(v, ictx)
			case *ProjectorStmt:
				analyzeProjector(v, ictx)
			case *JobStmt:
				analyzeJob(v, ictx)
			case *TableStmt:
				analyzeTable(v, ictx)
			case *TypeStmt:
				analyzeType(v, ictx)
			case *ViewStmt:
				analyzeView(v, ictx)
			case *UseWorkspaceStmt:
				analyzeUseWorkspace(v, ictx)
			case *StorageStmt:
				analyzeStorage(v, ictx)
			case *RoleStmt:
				analyzeRole(v, ictx)
			case *RateStmt:
				analyzeRate(v, ictx)
			case *LimitStmt:
				analyzeLimit(v, ictx)
			}
		})
	}
	// Pass 2
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*WorkspaceStmt); ok {
				analyzeWorkspace(v, ictx)
			}
		})
	}
	// Pass 3
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*AlterWorkspaceStmt); ok {
				analyzeAlterWorkspace(v, ictx)
				if v.alteredWorkspace != nil {
					includeChildWorkspaces(v, v.alteredWorkspace)
				}
			}
		})
	}
	// Pass 4
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*WorkspaceStmt); ok {
				includeFromInheritedWorkspaces(v, ictx)
				includeChildWorkspaces(v, v)
			}
		})
	}
	// Pass 5
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*UseWorkspaceStmt); ok {
				analyzeUsedWorkspaces(v, ictx)
			}
		})
	}
	// Pass 6
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			switch v := stmt.(type) {
			case *GrantStmt:
				analyseGrant(v, ictx)
			case *RevokeStmt:
				analyseRevoke(v, ictx)
			case *TableStmt:
				analyseRefFields(v.Items, ictx)
			case *TypeStmt:
				analyseRefFields(v.Items, ictx)
			case *ViewStmt:
				analyseViewRefFields(v.Items, ictx)
			}
		})
	}
}

func analyseGrantOrRevoke(toOrFrom DefQName, grant *GrantOrRevoke, c *iterateCtx) {
	// To
	err := resolveInCtx(toOrFrom, c, func(f *RoleStmt, pkg *PackageSchemaAST) error {
		grant.toRole = pkg.NewQName(f.Name)
		return nil
	})
	if err != nil {
		c.stmtErr(&toOrFrom.Pos, err)
	}
	// Role
	if grant.Role != nil {
		err = resolveInCtx(*grant.Role, c, func(role *RoleStmt, pkg *PackageSchemaAST) error {
			grant.Role.qName = pkg.NewQName(role.Name)
			grant.ops = append(grant.ops, appdef.OperationKind_Inherits)
			return nil
		})
		if err != nil {
			c.stmtErr(&grant.Role.Pos, err)
		}
	}
	// EXECUTE ON COMMAND
	if grant.Command != nil {
		err = resolveInCtx(*grant.Command, c, func(cmd *CommandStmt, pkg *PackageSchemaAST) error {
			grant.Command.qName = pkg.NewQName(cmd.Name)
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
			return nil
		})
		if err != nil {
			c.stmtErr(&grant.Command.Pos, err)
		}
	}

	// EXECUTE ON QUERY
	if grant.Query != nil {
		err = resolveInCtx(*grant.Query, c, func(query *QueryStmt, pkg *PackageSchemaAST) error {
			grant.Query.qName = pkg.NewQName(query.Name)
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
			return nil
		})
		if err != nil {
			c.stmtErr(&grant.Query.Pos, err)
		}
	}

	// SELECT ON VIEW
	if grant.View != nil {
		err := resolveInCtx(grant.View.View, c, func(view *ViewStmt, pkg *PackageSchemaAST) error {
			grant.View.View.qName = pkg.NewQName(view.Name)
			grant.ops = append(grant.ops, appdef.OperationKind_Select)
			// check columns
			checkColumn := func(column Identifier) error {
				for _, f := range view.Items {
					if f.Field != nil && f.Field.Name.Value == column.Value {
						grant.columns = append(grant.columns, string(column.Value))
						return nil
					}
					if f.RefField != nil && f.RefField.Name.Value == column.Value {
						grant.columns = append(grant.columns, string(column.Value))
						return nil
					}
				}
				return ErrUndefinedField(string(column.Value))
			}
			for _, i := range grant.View.Columns {
				if err := checkColumn(i); err != nil {
					c.stmtErr(&i.Pos, err)
				}
			}
			return nil
		})
		if err != nil {
			c.stmtErr(&grant.View.View.Pos, err)
		}
	}

	// if grant.Workspace {
	// 	err := resolveInCtx(grant.On, c, func(f *WorkspaceStmt, _ *PackageSchemaAST) error { return nil })
	// 	if err != nil {
	// 		c.stmtErr(&grant.On.Pos, err)
	// 	}
	// }

	// ALL COMMANDS WITH TAG
	if grant.AllCommandsWithTag != nil {
		if err := resolveInCtx(*grant.AllCommandsWithTag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
			grant.AllCommandsWithTag.qName = tagPkg.NewQName(tag.Name)
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllCommandsWithTag.Pos, err)
		}
	}

	// ALL COMMANDS
	if grant.AllCommands {
		grant.ops = append(grant.ops, appdef.OperationKind_Execute)
	}

	// ALL QUERIES WITH TAG
	if grant.AllQueriesWithTag != nil {
		if err := resolveInCtx(*grant.AllQueriesWithTag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
			grant.AllQueriesWithTag.qName = tagPkg.NewQName(tag.Name)
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllQueriesWithTag.Pos, err)
		}
	}
	// ALL QUERIES
	if grant.AllQueries {
		grant.ops = append(grant.ops, appdef.OperationKind_Execute)
	}

	// ALL VIEWS WITH TAG
	if grant.AllViewsWithTag != nil {
		if err := resolveInCtx(*grant.AllViewsWithTag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			grant.ops = append(grant.ops, appdef.OperationKind_Select)
			grant.AllViewsWithTag.qName = tagPkg.NewQName(tag.Name)
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllViewsWithTag.Pos, err)
		}
	}

	// ALL VIEWS
	if grant.AllViews {
		grant.ops = append(grant.ops, appdef.OperationKind_Select)
	}

	// ALL TABLES WITH TAG
	if grant.AllTablesWithTag != nil {
		if err := resolveInCtx(grant.AllTablesWithTag.Tag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			for _, item := range grant.AllTablesWithTag.Items {
				if item.Insert {
					grant.ops = append(grant.ops, appdef.OperationKind_Insert)
				} else if item.Update {
					grant.ops = append(grant.ops, appdef.OperationKind_Update)
				} else if item.Select {
					grant.ops = append(grant.ops, appdef.OperationKind_Select)
				} else if item.Activate {
					grant.ops = append(grant.ops, appdef.OperationKind_Activate)
				} else if item.Deactivate {
					grant.ops = append(grant.ops, appdef.OperationKind_Deactivate)
				}
			}
			grant.AllTablesWithTag.Tag.qName = tagPkg.NewQName(tag.Name)
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllTablesWithTag.Tag.Pos, err)
		}
	}

	// ALL TABLES
	if grant.AllTables != nil {
		for _, item := range grant.AllTables.Items {
			if item.Insert {
				grant.ops = append(grant.ops, appdef.OperationKind_Insert)
			} else if item.Update {
				grant.ops = append(grant.ops, appdef.OperationKind_Update)
			} else if item.Select {
				grant.ops = append(grant.ops, appdef.OperationKind_Select)
			} else if item.Activate {
				grant.ops = append(grant.ops, appdef.OperationKind_Activate)
			} else if item.Deactivate {
				grant.ops = append(grant.ops, appdef.OperationKind_Deactivate)
			}
		}
	}

	// TABLE
	var named INamedStatement
	var items []TableItemExpr
	if grant.Table != nil {
		var table *TableStmt
		var descriptor *WsDescriptorStmt
		var pkg *PackageSchemaAST
		var err error

		err = resolveInCtx(grant.Table.Table, c, func(t *TableStmt, p *PackageSchemaAST) error {
			table = t
			pkg = p
			return nil
		})

		if err != nil && err.Error() == ErrUndefinedTable(grant.Table.Table).Error() {
			err = resolveInCtx(grant.Table.Table, c, func(d *WsDescriptorStmt, p *PackageSchemaAST) error {
				descriptor = d
				pkg = p
				return nil
			})
		}
		if err != nil {
			c.stmtErr(&grant.Table.Table.Pos, err)
			return
		}
		if table != nil {
			named = table
			items = table.Items
		} else {
			named = descriptor
			items = descriptor.Items
		}
		grant.Table.Table.qName = pkg.NewQName(Ident(named.GetName()))
		for _, item := range grant.Table.Items {
			if item.Insert {
				grant.ops = append(grant.ops, appdef.OperationKind_Insert)
			} else if item.Update {
				grant.ops = append(grant.ops, appdef.OperationKind_Update)
			} else if item.Select {
				grant.ops = append(grant.ops, appdef.OperationKind_Select)
			} else if item.Activate {
				grant.ops = append(grant.ops, appdef.OperationKind_Activate)
			} else if item.Deactivate {
				grant.ops = append(grant.ops, appdef.OperationKind_Deactivate)
			}
		}
		checkColumn := func(column Ident) error {
			for _, f := range items {
				if f.Field != nil && f.Field.Name == column {
					grant.columns = append(grant.columns, string(column))
					return nil
				}
				if f.RefField != nil && f.RefField.Name == column {
					grant.columns = append(grant.columns, string(column))
					return nil
				}
				if f.NestedTable != nil && f.NestedTable.Name == column {
					return nil
				}
			}
			return ErrUndefinedField(string(column))
		}
		if grant.Table.All != nil {
			for _, column := range grant.Table.All.Columns {
				if err := checkColumn(column.Value); err != nil {
					c.stmtErr(&column.Pos, err)
				}
			}
		}
		for _, i := range grant.Table.Items {
			for _, column := range i.Columns {
				if column.Name != nil {
					if err := checkColumn(column.Name.Value); err != nil {
						c.stmtErr(&column.Pos, err)
					}
				} else {
					grant.columns = append(grant.columns, column.SysName)
				}
			}
		}

	}

	grant.workspace = c.mustCurrentWorkspace()
}

func analyseGrant(grant *GrantStmt, c *iterateCtx) {
	analyseGrantOrRevoke(grant.To, &grant.GrantOrRevoke, c)
}

func analyseRevoke(revoke *RevokeStmt, c *iterateCtx) {
	analyseGrantOrRevoke(revoke.From, &revoke.GrantOrRevoke, c)
}

func analyzeUseWorkspace(u *UseWorkspaceStmt, c *iterateCtx) {
	u.workspace = c.mustCurrentWorkspace()
	resolveFunc := func(f *WorkspaceStmt, pkg *PackageSchemaAST) error {
		if f.Abstract {
			return ErrUseOfAbstractWorkspace(string(u.Workspace.Value))
		}
		u.useWs = &statementNode{Pkg: pkg, Stmt: f}
		return nil
	}
	err := resolveInCtx(DefQName{Package: Ident(c.pkg.Name), Name: u.Workspace.Value}, c, resolveFunc)
	if err != nil {
		c.stmtErr(&u.Workspace.Pos, err)
	}
}

func analyzeAlterWorkspace(u *AlterWorkspaceStmt, c *iterateCtx) {
	// find all included statements

	var iterTableItems func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr)
	iterTableItems = func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr) {
		for i := range items {
			if items[i].NestedTable != nil {
				iterTableItems(ws, wsctx, items[i].NestedTable.Table.Items)
			}
		}
	}

	var iter func(wsctx *wsCtx, coll IStatementCollection)
	iter = func(wsctx *wsCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			if collection, ok := stmt.(IStatementCollection); ok {
				if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
					iter(wsctx, collection)
				}
			}
			if t, ok := stmt.(*TableStmt); ok {
				iterTableItems(wsctx.ws, wsctx, t.Items)
			}
		})
	}
	if u.alteredWorkspace != nil {
		iter(c.wsCtxs[u.alteredWorkspace], u)
	}
}

func analyzeStorage(u *StorageStmt, c *iterateCtx) {
	if c.pkg.Path != appdef.SysPackage {
		c.stmtErr(&u.Pos, ErrStorageDeclaredOnlyInSys)
	}
}

func analyzeRate(r *RateStmt, c *iterateCtx) {

	if r.Value.Variable != nil {
		resolve := func(d *DeclareStmt, p *PackageSchemaAST) error {

			var count int32
			var resolved bool
			if c.variableResolver != nil {
				count, resolved = c.variableResolver.AsInt32(p.NewQName(d.Name))
			}
			if !resolved {
				count = d.DefaultValue
			}
			if count <= 0 {
				return ErrPositiveValueOnly
			}
			r.Value.count = uint32(count)
			return nil
		}
		if err := resolveInCtx(*r.Value.Variable, c, resolve); err != nil {
			c.stmtErr(&r.Value.Variable.Pos, err)
		}
	}
	r.workspace = c.mustCurrentWorkspace()
}

func analyzeLimit(limit *LimitStmt, c *iterateCtx) {
	err := resolveInCtx(limit.RateName, c, func(l *RateStmt, schema *PackageSchemaAST) error {
		limit.RateName.qName = schema.NewQName(l.Name)
		return nil
	})
	if err != nil {
		c.stmtErr(&limit.RateName.Pos, err)
	}
	allowedOps := func(ops appdef.OperationsSet) {
		if len(limit.Actions) == 0 {
			limit.ops = ops.AsArray()
			return
		}
		for _, op := range limit.Actions {
			if op.Execute {
				if !ops.Contains(appdef.OperationKind_Execute) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_EXECUTE))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Execute)
				}
			}
			if op.Insert {
				if !ops.Contains(appdef.OperationKind_Insert) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_INSERT))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Insert)
				}
			}
			if op.Activate {
				if !ops.Contains(appdef.OperationKind_Activate) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_ACTIVATE))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Activate)
				}
			}
			if op.Deactivate {
				if !ops.Contains(appdef.OperationKind_Deactivate) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_DEACTIVATE))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Deactivate)
				}
			}
			if op.Update {
				if !ops.Contains(appdef.OperationKind_Update) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_UPDATE))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Update)
				}
			}
			if op.Select {
				if !ops.Contains(appdef.OperationKind_Select) {
					c.stmtErr(&op.Pos, ErrLimitOperationNotAllowed(OP_SELECT))
				} else {
					limit.ops = append(limit.ops, appdef.OperationKind_Select)
				}
			}
		}
	}
	if limit.SingleItem != nil {
		if limit.SingleItem.Command != nil {
			if err = resolveInCtx(*limit.SingleItem.Command, c, func(t *CommandStmt, schema *PackageSchemaAST) error {
				limit.SingleItem.Command.qName = schema.NewQName(t.Name)
				allowedOps(set.From(appdef.OperationKind_Execute))
				return nil
			}); err != nil {
				c.stmtErr(&limit.SingleItem.Command.Pos, err)
			}
		}
		if limit.SingleItem.Query != nil {
			if err = resolveInCtx(*limit.SingleItem.Query, c, func(t *QueryStmt, schema *PackageSchemaAST) error {
				limit.SingleItem.Query.qName = schema.NewQName(t.Name)
				allowedOps(set.From(appdef.OperationKind_Execute))
				return nil
			}); err != nil {
				c.stmtErr(&limit.SingleItem.Query.Pos, err)
			}
		}
		if limit.SingleItem.View != nil {
			if err = resolveInCtx(*limit.SingleItem.View, c, func(t *ViewStmt, schema *PackageSchemaAST) error {
				limit.SingleItem.View.qName = schema.NewQName(t.Name)
				allowedOps(set.From(appdef.OperationKind_Select))
				return nil
			}); err != nil {
				c.stmtErr(&limit.SingleItem.View.Pos, err)
			}
		}
		if limit.SingleItem.Table != nil {
			if err = resolveInCtx(*limit.SingleItem.Table, c, func(t *TableStmt, schema *PackageSchemaAST) error {
				limit.SingleItem.Table.qName = schema.NewQName(t.Name)
				allowedOps(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate))
				return nil
			}); err != nil {
				c.stmtErr(&limit.SingleItem.Table.Pos, err)
			}
		}
	}

	if limit.AllItems != nil {
		if limit.AllItems.Commands {
			allowedOps(set.From(appdef.OperationKind_Execute))
		} else if limit.AllItems.Queries {
			allowedOps(set.From(appdef.OperationKind_Execute))
		} else if limit.AllItems.Views {
			allowedOps(set.From(appdef.OperationKind_Select))
		} else {
			allowedOps(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate))
		}
		if limit.AllItems.WithTag != nil {
			if err = resolveInCtx(*limit.AllItems.WithTag, c, func(t *TagStmt, schema *PackageSchemaAST) error {
				limit.AllItems.WithTag.qName = schema.NewQName(t.Name)
				return nil
			}); err != nil {
				c.stmtErr(&limit.AllItems.WithTag.Pos, err)
			}
		}
	}

	if limit.EachItem != nil {
		if limit.EachItem.Commands {
			allowedOps(set.From(appdef.OperationKind_Execute))
		} else if limit.EachItem.Queries {
			allowedOps(set.From(appdef.OperationKind_Execute))
		} else if limit.EachItem.Views {
			allowedOps(set.From(appdef.OperationKind_Select))
		} else {
			allowedOps(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate))
		}
		if limit.EachItem.WithTag != nil {
			if err = resolveInCtx(*limit.EachItem.WithTag, c, func(t *TagStmt, schema *PackageSchemaAST) error {
				limit.EachItem.WithTag.qName = schema.NewQName(t.Name)
				return nil
			}); err != nil {
				c.stmtErr(&limit.EachItem.WithTag.Pos, err)
			}
		}
	}
	limit.workspace = c.mustCurrentWorkspace()
}

func analyzeView(view *ViewStmt, c *iterateCtx) {
	view.pkRef = nil
	fields := make(map[string]int)
	for i := range view.Items {
		fe := &view.Items[i]
		if fe.PrimaryKey != nil {
			if view.pkRef != nil {
				c.stmtErr(&fe.PrimaryKey.Pos, ErrPrimaryKeyRedefined)
			} else {
				view.pkRef = fe.PrimaryKey
			}
		}
		if fe.Field != nil {
			f := fe.Field
			if _, ok := fields[string(f.Name.Value)]; ok {
				c.stmtErr(&f.Name.Pos, ErrRedefined(string(f.Name.Value)))
			} else {
				fields[string(f.Name.Value)] = i
			}
		} else if fe.RefField != nil {
			rf := fe.RefField
			if _, ok := fields[string(rf.Name.Value)]; ok {
				c.stmtErr(&rf.Name.Pos, ErrRedefined(string(rf.Name.Value)))
			} else {
				fields[string(rf.Name.Value)] = i
			}
		} else if fe.RecordField != nil {
			if c.pkg.Path != appdef.SysPackage {
				c.stmtErr(&fe.Pos, ErrRecordFieldsOnlyInSys)
			}
			rf := fe.RecordField
			fields[string(rf.Name.Value)] = i
		}
	}
	if view.pkRef == nil {
		c.stmtErr(&view.Pos, ErrPrimaryKeyNotDefined)
		return
	}

	for _, pkf := range view.pkRef.PartitionKeyFields {
		index, ok := fields[string(pkf.Value)]
		if !ok {
			c.stmtErr(&pkf.Pos, ErrUndefinedField(string(pkf.Value)))
		}
		if view.Items[index].RecordField != nil {
			c.stmtErr(&pkf.Pos, ErrViewFieldRecord(string(pkf.Value)))
		}
		fld := view.Items[index].Field
		if fld != nil {
			if fld.Type.Varchar != nil {
				c.stmtErr(&pkf.Pos, ErrViewFieldVarchar(string(pkf.Value)))
			}
			if fld.Type.Bytes != nil {
				c.stmtErr(&pkf.Pos, ErrViewFieldBytes(string(pkf.Value)))
			}
		}
	}

	for ccIndex, ccf := range view.pkRef.ClusteringColumnsFields {
		fieldIndex, ok := fields[string(ccf.Value)]
		last := ccIndex == len(view.pkRef.ClusteringColumnsFields)-1
		if !ok {
			c.stmtErr(&ccf.Pos, ErrUndefinedField(string(ccf.Value)))
		}
		if view.Items[fieldIndex].RecordField != nil {
			c.stmtErr(&ccf.Pos, ErrViewFieldRecord(string(ccf.Value)))
		}
		fld := view.Items[fieldIndex].Field
		if fld != nil {
			if fld.Type.Varchar != nil && !last {
				c.stmtErr(&ccf.Pos, ErrVarcharFieldInCC(string(ccf.Value)))
			}
			if fld.Type.Bytes != nil && !last {
				c.stmtErr(&ccf.Pos, ErrBytesFieldInCC(string(ccf.Value)))
			}
		}
	}

	// ResultOf
	var job *JobStmt
	var projector *ProjectorStmt
	var err error
	var schema *PackageSchemaAST

	err = resolveInCtx(view.ResultOf, c, func(stmt *ProjectorStmt, pkg *PackageSchemaAST) error {
		projector = stmt
		schema = pkg
		return nil
	})
	if err != nil && err.Error() == ErrUndefinedProjector(view.ResultOf).Error() {
		err = resolveInCtx(view.ResultOf, c, func(stmt *JobStmt, pkg *PackageSchemaAST) error {
			job = stmt
			schema = pkg
			return nil
		})
	}

	if err != nil {
		c.stmtErr(&view.ResultOf.Pos, err)
		return
	}

	var intents []StateStorage

	if projector != nil {
		view.asResultOf = schema.NewQName(projector.Name)
		intents = projector.Intents
	} else {
		view.asResultOf = schema.NewQName(job.Name)
		intents = job.Intents
	}

	var intentForView *StateStorage
	for i := 0; i < len(intents) && intentForView == nil; i++ {
		var isView bool
		intent := intents[i]
		_ = resolveInCtx(intent.Storage, c, func(storage *StorageStmt, _ *PackageSchemaAST) error {
			isView = isView || storage.EntityView
			return nil
		}) // ignore error

		if isView {
			for _, entity := range intent.Entities {
				if entity.Name == view.Name && (entity.Package == Ident(c.pkg.Name) || entity.Package == Ident("")) {
					intentForView = &intents[i]
					break
				}
			}
		}
	}
	if intentForView == nil {
		c.stmtErr(&view.ResultOf.Pos, ErrStatementDoesNotDeclareViewIntent(projector.GetName(), view.GetName()))
		return
	}

	view.workspace = c.mustCurrentWorkspace()

	analyseWith(&view.With, view, c)
}

func analyzeImport(imp *ImportStmt, c *iterateCtx) {
	localPkgName := imp.GetLocalPkgName()
	if !isIdentifier(localPkgName) {
		c.stmtErr(&imp.Pos, ErrInvalidLocalPackageName(localPkgName))
		return
	}
	if localPkgName == c.pkg.Name {
		c.stmtErr(&imp.Pos, ErrLocalPackageNameConflict(localPkgName))
		return
	}
	if c.pkg.localNameToPkgPath == nil {
		c.pkg.localNameToPkgPath = make(map[string]string)
	}
	// check if local package name is already used in the package
	if pkgPath, ok := c.pkg.localNameToPkgPath[localPkgName]; ok {
		if pkgPath != imp.Name {
			c.stmtErr(&imp.Pos, ErrLocalPackageNameAlreadyUsed(localPkgName, pkgPath))
			return
		}
	} else {
		c.pkg.localNameToPkgPath[localPkgName] = imp.Name
	}
}

func analyzeTag(tag *TagStmt, c *iterateCtx) {
	tag.workspace = c.mustCurrentWorkspace()
}

func analyzeCommand(cmd *CommandStmt, c *iterateCtx) {

	resolve := func(qn DefQName) {
		typ, _, err := lookupInCtx[*TypeStmt](qn, c)
		if typ == nil && err == nil {
			tbl, _, err := lookupInCtx[*TableStmt](qn, c)
			if tbl == nil && err == nil {
				c.stmtErr(&qn.Pos, ErrUndefinedTypeOrTable(qn))
			}
		}
		if err != nil {
			c.stmtErr(&qn.Pos, err)
		}
	}

	if cmd.Param != nil && cmd.Param.Def != nil {
		resolve(*cmd.Param.Def)
	}
	if cmd.UnloggedParam != nil && cmd.UnloggedParam.Def != nil {
		typ, _, err := lookupInCtx[*TypeStmt](*cmd.UnloggedParam.Def, c)
		if typ == nil && err == nil {
			tbl, _, err := lookupInCtx[*TableStmt](*cmd.UnloggedParam.Def, c)
			if tbl == nil && err == nil {
				c.stmtErr(&cmd.UnloggedParam.Def.Pos, ErrUndefinedTypeOrTable(*cmd.UnloggedParam.Def))
			}
		}
		if err != nil {
			c.stmtErr(&cmd.UnloggedParam.Def.Pos, err)
		}
	}
	if cmd.Returns != nil && cmd.Returns.Def != nil {
		resolve(*cmd.Returns.Def)
	}
	analyseWith(&cmd.With, cmd, c)
	checkState(cmd.State, c, func(sc *StorageScope) bool { return sc.Commands })
	checkIntents(cmd.Intents, c, func(sc *StorageScope) bool { return sc.Commands })

	cmd.workspace = c.mustCurrentWorkspace()
}

func analyzeQuery(query *QueryStmt, c *iterateCtx) {
	if query.Param != nil && query.Param.Def != nil {
		if err := resolveInCtx(*query.Param.Def, c, func(*TypeStmt, *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&query.Param.Def.Pos, err)
		}

	}
	if query.Returns.Def != nil {
		if err := resolveInCtx(*query.Returns.Def, c, func(*TypeStmt, *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&query.Returns.Def.Pos, err)
		}
	}
	analyseWith(&query.With, query, c)
	checkState(query.State, c, func(sc *StorageScope) bool { return sc.Queries })

	query.workspace = c.mustCurrentWorkspace()
}

func analyzeRole(r *RoleStmt, c *iterateCtx) {
	r.workspace = c.mustCurrentWorkspace()
}

func checkStorageEntity(key *StateStorage, f *StorageStmt, c *iterateCtx) error {
	if f.EntityRecord {
		if len(key.Entities) == 0 {
			return ErrStorageRequiresEntity(key.Storage.String())
		}
		for _, entity := range key.Entities {
			resolveFunc := func(f *TableStmt, pkg *PackageSchemaAST) error {
				if f.Abstract {
					return ErrAbstractTableNotAlowedInProjectors(entity.String())
				}
				key.entityQNames = append(key.entityQNames, pkg.NewQName(entity.Name))
				return nil
			}
			if err2 := resolveInCtx(entity, c, resolveFunc); err2 != nil {
				return err2
			}
		}
	}
	if f.EntityView {
		if len(key.Entities) == 0 {
			return ErrStorageRequiresEntity(key.Storage.String())
		}
		for _, entity := range key.Entities {
			if err2 := resolveInCtx(entity, c, func(view *ViewStmt, pkg *PackageSchemaAST) error {
				key.entityQNames = append(key.entityQNames, pkg.NewQName(entity.Name))
				return nil
			}); err2 != nil {
				return err2
			}
		}
	}
	return nil
}

type checkScopeFunc func(sc *StorageScope) bool

func checkState(state []StateStorage, c *iterateCtx, scope checkScopeFunc) {
	for i := range state {
		key := &state[i]
		if err := resolveInCtx(key.Storage, c, func(f *StorageStmt, pkg *PackageSchemaAST) error {
			if e := checkStorageEntity(key, f, c); e != nil {
				return e
			}
			read := false
			for _, op := range f.Ops {
				if op.Get || op.GetBatch || op.Read {
					for i := range op.Scope {
						if scope(&op.Scope[i]) {
							read = true
							break
						}
					}
				}
			}
			if !read {
				return ErrStorageNotInState(key.Storage.String())
			}
			key.storageQName = pkg.NewQName(key.Storage.Name)
			return nil
		}); err != nil {
			c.stmtErr(&key.Storage.Pos, err)
		}
	}
}

func checkIntents(intents []StateStorage, c *iterateCtx, scope checkScopeFunc) {
	for i := range intents {
		key := &intents[i]
		if err := resolveInCtx(key.Storage, c, func(f *StorageStmt, pkg *PackageSchemaAST) error {
			if e := checkStorageEntity(key, f, c); e != nil {
				return e
			}
			read := false
			for _, op := range f.Ops {
				if op.Insert || op.Update {
					for i := range op.Scope {
						if scope(&op.Scope[i]) {
							read = true
							break
						}
					}
				}
			}
			if !read {
				return ErrStorageNotInIntents(key.Storage.String())
			}
			key.storageQName = pkg.NewQName(key.Storage.Name)
			return nil
		}); err != nil {
			c.stmtErr(&key.Storage.Pos, err)
		}
	}
}

func preAnalyseAlterWorkspace(u *AlterWorkspaceStmt, c *iterateCtx) {
	resolveFunc := func(w *WorkspaceStmt, schema *PackageSchemaAST) error {
		u.alteredWorkspace = w
		u.alteredWorkspacePkg = schema
		if !w.Alterable && schema != c.pkg {
			return ErrWorkspaceIsNotAlterable(u.Name.String())
		}
		return nil
	}
	err := resolveInCtx(u.Name, c, resolveFunc)
	if err != nil {
		c.stmtErr(&u.Name.Pos, err)
		return
	}
}

func analyzeProjector(prj *ProjectorStmt, c *iterateCtx) {
	for i := range prj.Triggers {
		trigger := &prj.Triggers[i]

		if trigger.CronSchedule != nil {
			c.stmtErr(&prj.Pos, ErrScheduledProjectorDeprecated)
		}

		for i := range trigger.QNames {
			defQName := &trigger.QNames[i]
			if len(trigger.TableActions) > 0 {

				wd, pkg, err := lookupInCtx[*WsDescriptorStmt](*defQName, c)
				if err != nil {
					c.stmtErr(&defQName.Pos, err)
					continue
				}
				if wd != nil {
					defQName.qName = pkg.NewQName(wd.Name)
					continue
				}

				resolveFunc := func(table *TableStmt, pkg *PackageSchemaAST) error {
					crecord := (table.Name == Ident(istructs.QNameCRecord.Entity()))
					wrecord := (table.Name == Ident(istructs.QNameWRecord.Entity()))
					sysDoc := (pkg.Path == appdef.SysPackage) && (crecord || wrecord)
					if table.Abstract && !sysDoc {
						return ErrAbstractTableNotAlowedInProjectors(defQName.String())
					}
					k, _, err := getTableTypeKind(table, pkg, c)
					if err != nil {
						return err
					}
					if k == appdef.TypeKind_ODoc {
						if trigger.activate() || trigger.deactivate() || trigger.update() {
							return ErrOnlyInsertForOdocOrORecord
						}
					}
					defQName.qName = pkg.NewQName(table.Name)
					return nil
				}
				if err := resolveInCtx(*defQName, c, resolveFunc); err != nil {
					c.stmtErr(&defQName.Pos, err)
				}
			} else { // Command
				if trigger.ExecuteAction.WithParam {
					var pkg *PackageSchemaAST
					var odoc *TableStmt
					typ, pkg, err := lookupInCtx[*TypeStmt](*defQName, c)
					if err != nil { // type?
						c.stmtErr(&defQName.Pos, err)
						continue
					}
					if typ == nil { // ODoc?
						odoc, pkg, err = lookupInCtx[*TableStmt](*defQName, c)
						if err != nil {
							c.stmtErr(&defQName.Pos, err)
							continue
						}
						if odoc == nil || odoc.tableTypeKind != appdef.TypeKind_ODoc {
							c.stmtErr(&defQName.Pos, ErrUndefinedTypeOrOdoc(*defQName))
							continue
						}
					}
					defQName.qName = pkg.NewQName(defQName.Name)
				} else {
					err := resolveInCtx(*defQName, c, func(f *CommandStmt, pkg *PackageSchemaAST) error {
						defQName.qName = pkg.NewQName(f.Name)
						return nil
					})
					if err != nil {
						c.stmtErr(&defQName.Pos, err)
						continue
					}
				}
			}
		}
	}

	checkState(prj.State, c, func(sc *StorageScope) bool { return sc.Projectors })
	checkIntents(prj.Intents, c, func(sc *StorageScope) bool { return sc.Projectors })

	prj.workspace = c.mustCurrentWorkspace()
}

func analyzeJob(j *JobStmt, c *iterateCtx) {
	ws := getCurrentWorkspace(c)
	if ws.workspace == nil {
		panic("workspace not found for JOB" + j.Name)
	}
	if ws.workspace.GetName() != nameAppWorkspaceWS || ws.pkg.Name != appdef.SysPackage {
		c.stmtErr(&j.Pos, ErrJobMustBeInAppWorkspace)
	}

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, e := parser.Parse(*j.CronSchedule); e != nil {
		c.stmtErr(&j.Pos, ErrInvalidCronSchedule(*j.CronSchedule))
	}
	checkState(j.State, c, func(sc *StorageScope) bool { return sc.Jobs })
	checkIntents(j.Intents, c, func(sc *StorageScope) bool { return sc.Jobs })

	j.workspace = c.mustCurrentWorkspace()
}

// Note: function may update with argument
func analyseWith(with *[]WithItem, statement IStatement, c *iterateCtx) {
	var comment *WithItem

	for i := range *with {
		item := &(*with)[i]
		if item.Comment != nil {
			comment = item
		}
		for j := range item.Tags {
			tag := item.Tags[j]
			if err := resolveInCtx(tag, c, func(t *TagStmt, tPkg *PackageSchemaAST) error {
				qname := tPkg.NewQName(t.Name)
				item.tags = append(item.tags, qname)
				return nil
			}); err != nil {
				c.stmtErr(&tag.Pos, err)
			}
		}
	}

	if comment != nil {
		statement.SetComments(strings.Split(*comment.Comment, "\n"))
	}
}

func preAnalyseTable(v *TableStmt, c *iterateCtx) {
	var err error
	v.tableTypeKind, v.singleton, err = getTableTypeKind(v, c.pkg, c)
	if err != nil {
		c.stmtErr(&v.Pos, err)
		return
	}
}

func analyzeTable(v *TableStmt, c *iterateCtx) {
	analyseWith(&v.With, v, c)
	analyseNestedTables(v.Items, v.tableTypeKind, c)
	analyseFieldSets(v.Items, c)
	analyseFields(v.Items, c, true)
	if v.Inherits != nil {
		resolvedFunc := func(f *TableStmt, p *PackageSchemaAST) error {
			if !f.Abstract {
				return ErrBaseTableMustBeAbstract
			}
			v.inherits = tableAddr{f, p}
			return nil
		}
		if err := resolveInCtx(*v.Inherits, c, resolvedFunc); err != nil {
			c.stmtErr(&v.Inherits.Pos, err)
		}
	}
	v.workspace = c.mustCurrentWorkspace()
}

func analyzeType(v *TypeStmt, c *iterateCtx) {
	for _, i := range v.Items {
		if i.NestedTable != nil {
			c.stmtErr(&i.NestedTable.Pos, ErrNestedTablesNotSupportedInTypes)
		}
	}
	analyseFieldSets(v.Items, c)
	analyseFields(v.Items, c, false)
	v.workspace = c.mustCurrentWorkspace()
}

func analyzeWorkspace(v *WorkspaceStmt, c *iterateCtx) {

	wsc := &wsCtx{
		pkg:  c.pkg,
		ws:   v,
		ictx: c,
	}
	c.wsCtxs[v] = wsc

	var chain []DefQName
	var checkChain func(qn DefQName) error

	checkChain = func(qn DefQName) error {
		resolveFunc := func(w *WorkspaceStmt, wp *PackageSchemaAST) error {
			if !w.Abstract {
				return ErrBaseWorkspaceMustBeAbstract
			}
			for i := range chain {
				if chain[i] == qn {
					return ErrCircularReferenceInInherits
				}
			}
			chain = append(chain, qn)
			for _, w := range w.Inherits {
				e := checkChain(w)
				if e != nil {
					return e
				}
			}
			v.inheritedWorkspaces = append(v.inheritedWorkspaces, w)
			return nil
		}
		return resolveInCtx(qn, c, resolveFunc)
	}

	for _, inherits := range v.Inherits {
		chain = make([]DefQName, 0)
		if err := checkChain(inherits); err != nil {
			c.stmtErr(&inherits.Pos, err)
			return
		}
	}
	if v.Descriptor != nil {
		wc := &iterateCtx{
			basicContext: c.basicContext,
			collection:   v,
			pkg:          c.pkg,
			parent:       c,
			wsCtxs:       c.wsCtxs,
		}
		if v.Abstract {
			c.stmtErr(&v.Descriptor.Pos, ErrAbstractWorkspaceDescriptor)
		}
		analyseNestedTables(v.Descriptor.Items, appdef.TypeKind_CDoc, wc)
		analyseFields(v.Descriptor.Items, wc, true)
		analyseFieldSets(v.Descriptor.Items, wc)
		v.Descriptor.workspace = workspaceAddr{v, c.pkg}
	}

	// GRANT shall not follow REVOKE
	revokeFound := false
	for _, s := range v.Statements {
		if s.Revoke != nil {
			revokeFound = true
		}
		if s.Grant != nil && revokeFound {
			c.stmtErr(&s.Grant.Pos, ErrGrantFollowsRevoke)
		}

	}
}

type wsCtx struct {
	pkg  *PackageSchemaAST
	ws   *WorkspaceStmt
	ictx *iterateCtx
}

func includeFromInheritedWorkspaces(ws *WorkspaceStmt, c *iterateCtx) {
	sysWorkspace, err := lookupInSysPackage(c.basicContext, DefQName{Package: appdef.SysPackage, Name: rootWorkspaceName})
	if err != nil {
		c.stmtErr(&ws.Pos, err)
		return
	}
	var addFromInheritedWs func(ws *WorkspaceStmt, wsctx *wsCtx)
	var added []*WorkspaceStmt

	addFromInheritedWs = func(ws *WorkspaceStmt, wsctx *wsCtx) {
		added = append(added, ws)
		for _, inherits := range ws.Inherits {
			var baseWs *WorkspaceStmt
			err := resolveInCtx(inherits, wsctx.ictx, func(ws *WorkspaceStmt, _ *PackageSchemaAST) error {
				baseWs = ws
				if baseWs == sysWorkspace {
					return ErrInheritanceFromSysWorkspaceNotAllowed
				}
				return nil
			})
			if err != nil {
				c.stmtErr(&ws.Pos, err)
				return
			}
			for _, item := range added {
				if item == baseWs {
					return // circular reference
				}
			}

			addFromInheritedWs(baseWs, wsctx)
			added = append(added, baseWs)
		}
	}
	addFromInheritedWs(ws, c.wsCtxs[ws])
}

func includeChildWorkspaces(collection IStatementCollection, ws *WorkspaceStmt) {
	collection.Iterate(func(stmt interface{}) {
		if child, ok := stmt.(*WorkspaceStmt); ok {
			ws.usedWorkspaces = append(ws.usedWorkspaces, child)
		}
	})
}

func analyzeUsedWorkspaces(uws *UseWorkspaceStmt, _ *iterateCtx) {
	if ws := uws.workspace.workspace; ws != nil && uws.useWs != nil {
		if usedWS, ok := uws.useWs.Stmt.(*WorkspaceStmt); ok {
			ws.usedWorkspaces = append(ws.usedWorkspaces, usedWS)
		}
	}
}

func analyseNestedTables(items []TableItemExpr, rootTableKind appdef.TypeKind, c *iterateCtx) {
	for i := range items {
		item := items[i]

		var nestedTable *TableStmt
		var pos *lexer.Position

		if item.NestedTable != nil {
			nestedTable = &item.NestedTable.Table
			pos = &item.NestedTable.Pos

			if nestedTable.Abstract {
				c.stmtErr(pos, ErrNestedAbstractTable(nestedTable.GetName()))
				return
			}
			if nestedTable.Inherits == nil {
				var err error
				nestedTable.tableTypeKind, err = getNestedTableKind(rootTableKind)
				if err != nil {
					c.stmtErr(pos, err)
					return
				}
			} else {
				var err error
				nestedTable.tableTypeKind, nestedTable.singleton, err = getTableTypeKind(nestedTable, c.pkg, c)
				if err != nil {
					c.stmtErr(pos, err)
					return
				}
				tk, err := getNestedTableKind(rootTableKind)
				if err != nil {
					c.stmtErr(pos, err)
					return
				}
				if nestedTable.tableTypeKind != tk {
					c.stmtErr(pos, ErrNestedTableIncorrectKind)
					return
				}
			}
			nestedTable.workspace = getCurrentWorkspace(c)
			analyseWith(&nestedTable.With, nestedTable, c)
			analyseNestedTables(nestedTable.Items, rootTableKind, c)

		}
	}
}

func analyseFieldSets(items []TableItemExpr, c *iterateCtx) {
	for i := range items {
		item := items[i]
		if item.FieldSet != nil {
			if err := resolveInCtx(item.FieldSet.Type, c, func(t *TypeStmt, _ *PackageSchemaAST) error {
				item.FieldSet.typ = t
				return nil
			}); err != nil {
				c.stmtErr(&item.FieldSet.Type.Pos, err)
				continue
			}
		}
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			analyseFieldSets(nestedTable.Items, c)
		}
	}
}

func lookupField(items []TableItemExpr, name Ident, c *iterateCtx) (found bool) {
	for i := range items {
		item := items[i]
		if item.Field != nil {
			if item.Field.Name == name {
				return true
			}
		}
		if item.FieldSet != nil {
			if err := resolveInCtx(item.FieldSet.Type, c, func(t *TypeStmt, schema *PackageSchemaAST) error {
				found = lookupField(t.Items, name, c)
				return nil
			}); err != nil {
				c.stmtErr(&item.FieldSet.Pos, err)
				return false
			}
		}
	}
	return found
}

func analyseFields(items []TableItemExpr, c *iterateCtx, isTable bool) {
	fieldsInUniques := make([]Ident, 0)
	constraintNames := make(map[string]bool)
	for i := range items {
		item := items[i]
		if item.Field != nil {
			field := item.Field
			if field.CheckRegexp != nil {
				if field.Type.DataType != nil && field.Type.DataType.Varchar != nil {
					_, err := regexp.Compile(field.CheckRegexp.Regexp)
					if err != nil {
						c.stmtErr(&field.CheckRegexp.Pos, ErrCheckRegexpErr(err))
					}
				} else {
					c.stmtErr(&field.CheckRegexp.Pos, ErrRegexpCheckOnlyForVarcharField)
				}
			}
			if field.Type.DataType != nil {
				vc := field.Type.DataType.Varchar
				if vc != nil && vc.MaxLen != nil {
					if *vc.MaxLen > uint64(appdef.MaxFieldLength) {
						c.stmtErr(&vc.Pos, ErrMaxFieldLengthTooLarge)
					}
				}
				bb := field.Type.DataType.Bytes
				if bb != nil && bb.MaxLen != nil {
					if *bb.MaxLen > uint64(appdef.MaxFieldLength) {
						c.stmtErr(&bb.Pos, ErrMaxFieldLengthTooLarge)
					}
				}
			} else {
				if !isTable { // analysing a TYPE
					err := resolveInCtx(*field.Type.Def, c, func(f *TypeStmt, pkg *PackageSchemaAST) error {
						field.Type.qName = pkg.NewQName(f.Name)
						return nil
					})
					if err != nil {
						c.stmtErr(&field.Type.Def.Pos, err)
						continue
					}
				} else { // analysing a TABLE
					err := resolveInCtx(*field.Type.Def, c, func(f *TableStmt, pkg *PackageSchemaAST) error {
						if f.Abstract {
							return ErrNestedAbstractTable(field.Type.Def.String())
						}
						if f.tableTypeKind != appdef.TypeKind_CRecord && f.tableTypeKind != appdef.TypeKind_ORecord && f.tableTypeKind != appdef.TypeKind_WRecord {
							return ErrTypeNotSupported(field.Type.Def.String())
						}
						field.Type.qName = pkg.NewQName(f.Name)
						field.Type.tableStmt = f
						field.Type.tablePkg = pkg
						return nil
					})
					if err != nil {
						if err.Error() == ErrUndefinedTable(*field.Type.Def).Error() {
							c.stmtErr(&field.Type.Def.Pos, ErrUndefinedDataTypeOrTable(*field.Type.Def))
						} else {
							c.stmtErr(&field.Type.Def.Pos, err)
						}
						continue
					}
				}
			}
		}
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			analyseFields(nestedTable.Items, c, true)
		}
		if item.Constraint != nil {
			if item.Constraint.ConstraintName != "" {
				cname := string(item.Constraint.ConstraintName)
				if _, ok := constraintNames[cname]; ok {
					c.stmtErr(&item.Constraint.Pos, ErrRedefined(cname))
					continue
				}
				constraintNames[cname] = true
			}
			if item.Constraint.UniqueField != nil {
				if ok := lookupField(items, item.Constraint.UniqueField.Field, c); !ok {
					c.stmtErr(&item.Constraint.Pos, ErrUndefinedField(string(item.Constraint.UniqueField.Field)))
					continue
				}
			} else if item.Constraint.Unique != nil {
				for _, field := range item.Constraint.Unique.Fields {
					for _, f := range fieldsInUniques {
						if f == field {
							c.stmtErr(&item.Constraint.Pos, ErrFieldAlreadyInUnique(string(field)))
							continue
						}
					}
					if ok := lookupField(items, field, c); !ok {
						c.stmtErr(&item.Constraint.Pos, ErrUndefinedField(string(field)))
						continue
					}
					fieldsInUniques = append(fieldsInUniques, field)
				}
			}
		}
	}
}

func analyseRefFields(items []TableItemExpr, c *iterateCtx) {
	for i := range items {
		item := items[i]
		if item.RefField != nil {
			rf := item.RefField
			for i := range rf.RefDocs {
				if err := resolveInCtx(rf.RefDocs[i], c, func(f *TableStmt, tblPkg *PackageSchemaAST) error {
					if f.Abstract {
						return ErrReferenceToAbstractTable(rf.RefDocs[i].String())
					}
					rf.refQNames = append(rf.refQNames, tblPkg.NewQName(f.Name))
					rf.refTables = append(rf.refTables, tableAddr{f, tblPkg})
					return nil
				}); err != nil {
					c.stmtErr(&rf.RefDocs[i].Pos, err)
					continue
				}
			}
		}
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			analyseRefFields(nestedTable.Items, c)
		}
	}
}

func analyseViewRefFields(items []ViewItemExpr, c *iterateCtx) {
	for i := range items {
		item := items[i]
		if item.RefField != nil {
			rf := item.RefField
			for i := range rf.RefDocs {
				if err := resolveInCtx(rf.RefDocs[i], c, func(f *TableStmt, tblPkg *PackageSchemaAST) error {
					if f.Abstract {
						return ErrReferenceToAbstractTable(rf.RefDocs[i].String())
					}
					return nil
				}); err != nil {
					c.stmtErr(&rf.RefDocs[i].Pos, err)
					continue
				}
			}
		}
	}
}

type tableNode struct {
	pkg   *PackageSchemaAST
	table *TableStmt
}

func getTableInheritanceChain(table *TableStmt, c *iterateCtx) (chain []tableNode, err error) {
	chain = make([]tableNode, 0)
	refCycle := func(node tableNode) bool {
		for i := range chain {
			if (chain[i].pkg == node.pkg) && (chain[i].table.Name == node.table.Name) {
				return true
			}
		}
		return false
	}
	var vf func(t *TableStmt) error
	vf = func(t *TableStmt) error {
		if t.Inherits != nil {
			inherited := *t.Inherits
			t, pkg, err := lookupInCtx[*TableStmt](inherited, c)
			if err != nil {
				return err
			}
			if t != nil {
				node := tableNode{pkg: pkg, table: t}
				if refCycle(node) {
					return ErrCircularReferenceInInherits
				}
				chain = append(chain, node)
				return vf(t)
			}
		}
		return nil
	}
	err = vf(table)
	return
}

func getTableTypeKind(table *TableStmt, pkg *PackageSchemaAST, c *iterateCtx) (kind appdef.TypeKind, singleton bool, err error) {

	kind = appdef.TypeKind_null
	check := func(node tableNode) {
		if node.pkg.Path == appdef.SysPackage {
			switch node.table.Name {
			case Ident(istructs.QNameCRecord.Entity()):
				kind = appdef.TypeKind_CRecord
			case Ident(istructs.QNameORecord.Entity()):
				kind = appdef.TypeKind_ORecord
			case Ident(istructs.QNameWRecord.Entity()):
				kind = appdef.TypeKind_WRecord
			case Ident(istructs.QNameCDoc.Entity()):
				kind = appdef.TypeKind_CDoc
			case Ident(istructs.QNameODoc.Entity()):
				kind = appdef.TypeKind_ODoc
			case Ident(istructs.QNameWDoc.Entity()):
				kind = appdef.TypeKind_WDoc
			case nameCSingleton:
				kind = appdef.TypeKind_CDoc
				singleton = true
			case nameWSingleton:
				kind = appdef.TypeKind_WDoc
				singleton = true
			}
		}
	}

	check(tableNode{pkg: pkg, table: table})
	if kind != appdef.TypeKind_null {
		return kind, singleton, nil
	}

	chain, e := getTableInheritanceChain(table, c)
	if e != nil {
		return appdef.TypeKind_null, false, e
	}
	for _, t := range chain {
		check(t)
		if kind != appdef.TypeKind_null {
			return kind, singleton, nil
		}
	}
	return appdef.TypeKind_null, false, ErrUndefinedTableKind
}
