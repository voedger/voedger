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
)

type iterateCtx struct {
	*basicContext
	pkg        *PackageSchemaAST
	collection IStatementCollection
	parent     *iterateCtx
	wsCtxs     map[*WorkspaceStmt]*wsCtx
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
			case *CommandStmt:
				analyzeCommand(v, ictx)
			case *QueryStmt:
				analyzeQuery(v, ictx)
			case *ProjectorStmt:
				analyseProjector(v, ictx)
			case *JobStmt:
				analyseJob(v, ictx)
			case *TableStmt:
				analyseTable(v, ictx)
			case *TypeStmt:
				analyseType(v, ictx)
			case *ViewStmt:
				analyseView(v, ictx)
			case *UseTableStmt:
				analyseUseTable(v, ictx)
			case *UseWorkspaceStmt:
				analyseUseWorkspace(v, ictx)
			case *StorageStmt:
				analyseStorage(v, ictx)
			case *RateStmt:
				analyseRate(v, ictx)
			case *LimitStmt:
				analyseLimit(v, ictx)
			}
		})
	}
	// Pass 2
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*WorkspaceStmt); ok {
				analyseWorkspace(v, ictx)
			}
		})
	}
	// Pass 3
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*AlterWorkspaceStmt); ok {
				analyseAlterWorkspace(v, ictx)
			}
		})
	}
	// Pass 4
	for _, p := range packages {
		ictx.setPkg(p)
		iterateContext(ictx, func(stmt interface{}, ictx *iterateCtx) {
			if v, ok := stmt.(*WorkspaceStmt); ok {
				includeFromInheritedWorkspaces(v, ictx)
			}
		})
	}
	// Pass 5
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

func iterateWorkspaceStmts[stmtType INamedStatement](ctx *iterateCtx, onlyCurrentWs bool, callback func(stmt stmtType, schema *PackageSchemaAST, ctx *iterateCtx)) {
	currentWs := getCurrentWorkspace(ctx)
	for _, stmt := range currentWs.nodes {
		if onlyCurrentWs && currentWs != stmt.workspace {
			continue
		}
		if s, ok := stmt.node.Stmt.(stmtType); ok {
			callback(s, stmt.node.Pkg, ctx)
		}
	}
}

func resolveInCurrentWs[stmtType *CommandStmt | *QueryStmt | *ViewStmt | *TableStmt | *RoleStmt](qn DefQName, ctx *iterateCtx) (result stmtType, pkg *PackageSchemaAST, err error) {
	err = resolveInCtx(qn, ctx, func(f stmtType, p *PackageSchemaAST) error {
		currentWs := getCurrentWorkspace(ctx)
		for _, n := range currentWs.nodes {
			if n.node.Stmt.GetName() == string(qn.Name) && n.node.Pkg == p {
				result = f
				pkg = n.node.Pkg
				return nil
			}
		}
		return nil
	})
	return result, pkg, err
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
		role, pkg, err := resolveInCurrentWs[*RoleStmt](*grant.Role, c)
		if err != nil {
			c.stmtErr(&grant.Role.Pos, err)
		} else if role != nil {
			grant.on = append(grant.on, pkg.NewQName(role.Name))
			grant.ops = append(grant.ops, appdef.OperationKind_Inherits)
		} else {
			c.stmtErr(&grant.Role.Pos, ErrUndefinedRole(*grant.Role))
		}
	}
	// INSERT ON COMMAND
	if grant.Command != nil {
		cmd, pkg, err := resolveInCurrentWs[*CommandStmt](*grant.Command, c)
		if err != nil {
			c.stmtErr(&grant.Command.Pos, err)
		} else if cmd != nil {
			grant.on = append(grant.on, pkg.NewQName(cmd.Name))
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
		} else {
			c.stmtErr(&grant.Command.Pos, ErrUndefinedCommand(*grant.Command))
		}
	}

	// SELECT ON QUERY
	if grant.Query != nil {
		query, pkg, err := resolveInCurrentWs[*QueryStmt](*grant.Query, c)
		if err != nil {
			c.stmtErr(&grant.Query.Pos, err)
		} else if query != nil {
			grant.on = append(grant.on, pkg.NewQName(query.Name))
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
		} else {
			c.stmtErr(&grant.Query.Pos, ErrUndefinedQuery(*grant.Query))
		}
	}

	// SELECT ON VIEW
	if grant.View != nil {
		view, pkg, err := resolveInCurrentWs[*ViewStmt](grant.View.View, c)
		if err != nil {
			c.stmtErr(&grant.View.View.Pos, err)
		} else if view != nil {
			grant.on = append(grant.on, pkg.NewQName(view.Name))
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
		} else {
			c.stmtErr(&grant.View.View.Pos, ErrUndefinedView(grant.View.View))
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
			iterateWorkspaceStmts(c, false, func(cmd *CommandStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
				if hasTags(cmd.With, tag, c) {
					grant.on = append(grant.on, schema.NewQName(cmd.Name))
				}
			})
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllCommandsWithTag.Pos, err)
		}
	}

	// ALL COMMANDS
	if grant.AllCommands {
		grant.ops = append(grant.ops, appdef.OperationKind_Execute)
		iterateWorkspaceStmts(c, true, func(cmd *CommandStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
			grant.on = append(grant.on, schema.NewQName(cmd.Name))
		})
	}

	// ALL QUERIES WITH TAG
	if grant.AllQueriesWithTag != nil {
		if err := resolveInCtx(*grant.AllQueriesWithTag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			grant.ops = append(grant.ops, appdef.OperationKind_Execute)
			iterateWorkspaceStmts(c, false, func(query *QueryStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
				if hasTags(query.With, tag, c) {
					grant.on = append(grant.on, schema.NewQName(query.Name))
				}
			})
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllQueriesWithTag.Pos, err)
		}
	}

	// ALL QUERIES
	if grant.AllQueries {
		grant.ops = append(grant.ops, appdef.OperationKind_Execute)
		iterateWorkspaceStmts(c, true, func(query *QueryStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
			grant.on = append(grant.on, schema.NewQName(query.Name))
		})
	}

	// ALL VIEWS WITH TAG
	if grant.AllViewsWithTag != nil {
		if err := resolveInCtx(*grant.AllViewsWithTag, c, func(tag *TagStmt, tagPkg *PackageSchemaAST) error {
			grant.ops = append(grant.ops, appdef.OperationKind_Select)
			iterateWorkspaceStmts(c, false, func(view *ViewStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
				if hasTags(view.With, tag, c) {
					grant.on = append(grant.on, schema.NewQName(view.Name))
				}
			})
			return nil
		}); err != nil {
			c.stmtErr(&grant.AllViewsWithTag.Pos, err)
		}
	}

	// ALL VIEWS
	if grant.AllViews {
		grant.ops = append(grant.ops, appdef.OperationKind_Select)
		iterateWorkspaceStmts(c, true, func(view *ViewStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
			grant.on = append(grant.on, schema.NewQName(view.Name))
		})
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
				}
			}
			iterateWorkspaceStmts(c, false, func(tbl *TableStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
				if hasTags(tbl.With, tag, c) {
					grant.on = append(grant.on, schema.NewQName(tbl.Name))
				}
			})
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
			}
		}
		iterateWorkspaceStmts(c, true, func(tbl *TableStmt, schema *PackageSchemaAST, ctx *iterateCtx) {
			grant.on = append(grant.on, schema.NewQName(tbl.Name))
		})
	}

	// TABLE
	if grant.Table != nil {
		table, pkg, err := resolveInCurrentWs[*TableStmt](grant.Table.Table, c)
		if err != nil {
			c.stmtErr(&grant.Table.Table.Pos, err)
		} else if table != nil {
			grant.on = append(grant.on, pkg.NewQName(table.Name))
			for _, item := range grant.Table.Items {
				if item.Insert {
					grant.ops = append(grant.ops, appdef.OperationKind_Insert)
				} else if item.Update {
					grant.ops = append(grant.ops, appdef.OperationKind_Update)
				} else if item.Select {
					grant.ops = append(grant.ops, appdef.OperationKind_Select)
				}
			}
			checkColumn := func(column Ident) error {
				for _, f := range table.Items {
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
					if err := checkColumn(column.Value); err != nil {
						c.stmtErr(&column.Pos, err)
					}
				}
			}
		} else {
			c.stmtErr(&grant.Table.Table.Pos, ErrUndefinedTable(grant.Table.Table))
		}
	}

}

func hasTags(with []WithItem, tag *TagStmt, c *iterateCtx) bool {
	for _, w := range with {
		for _, t := range w.Tags {
			if t.Name == tag.Name {
				withTag, _, err := lookupInCtx[*TagStmt](t, c)
				if err == nil && withTag == tag {
					return true
				}
			}
		}
	}
	return false
}

func analyseGrant(grant *GrantStmt, c *iterateCtx) {
	analyseGrantOrRevoke(grant.To, &grant.GrantOrRevoke, c)
}

func analyseRevoke(revoke *RevokeStmt, c *iterateCtx) {
	analyseGrantOrRevoke(revoke.From, &revoke.GrantOrRevoke, c)
}

func analyseUseTable(u *UseTableStmt, c *iterateCtx) {

	var pkg *PackageSchemaAST
	var pkgName Ident = ""
	var err error

	if u.Package != nil {
		pkg, err = findPackage(u.Package.Value, c)
		if err != nil {
			c.stmtErr(&u.Package.Pos, err)
			return
		}
		pkgName = u.Package.Value
	}

	if u.AllTables {
		var iter func(tbl *TableStmt)
		iter = func(tbl *TableStmt) {
			if !tbl.Abstract {
				u.registerQName(pkg.NewQName(tbl.Name), statementNode{Pkg: pkg, Stmt: tbl})
			}
			for _, item := range tbl.Items {
				if item.NestedTable != nil {
					iter(&item.NestedTable.Table)
				}
			}

		}
		if pkg == nil {
			pkg = c.pkg
		}
		for _, stmt := range pkg.Ast.Statements {
			if stmt.Table != nil {
				iter(stmt.Table)
			}
		}
	} else {
		defQName := DefQName{Package: pkgName, Name: u.TableName.Value}
		err = resolveInCtx(defQName, c, func(tbl *TableStmt, pkg *PackageSchemaAST) error {
			if tbl.Abstract {
				return ErrUseOfAbstractTable(defQName.String())
			}
			u.registerQName(pkg.NewQName(tbl.Name), statementNode{Pkg: pkg, Stmt: tbl})
			return nil
		})
		if err != nil {
			c.stmtErr(&u.TableName.Pos, err)
		}
	}
}

func analyseUseWorkspace(u *UseWorkspaceStmt, c *iterateCtx) {
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

func analyseAlterWorkspace(u *AlterWorkspaceStmt, c *iterateCtx) {
	// find all included statements

	var iterTableItems func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr)
	iterTableItems = func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr) {
		for i := range items {
			if items[i].NestedTable != nil {
				useStmtInWs(wsctx, wsctx.pkg, &items[i].NestedTable.Table)
				iterTableItems(ws, wsctx, items[i].NestedTable.Table.Items)
			}
		}
	}

	var iter func(wsctx *wsCtx, coll IStatementCollection)
	iter = func(wsctx *wsCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			useStmtInWs(wsctx, c.pkg, stmt)
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

func analyseStorage(u *StorageStmt, c *iterateCtx) {
	if c.pkg.Path != appdef.SysPackage {
		c.stmtErr(&u.Pos, ErrStorageDeclaredOnlyInSys)
	}
}

func analyseRate(r *RateStmt, c *iterateCtx) {
	if r.Value.Variable != nil {
		resolved := func(d *DeclareStmt, p *PackageSchemaAST) error {
			r.Value.variable = p.NewQName(d.Name)
			r.Value.declare = d
			return nil
		}
		if err := resolveInCtx(*r.Value.Variable, c, resolved); err != nil {
			c.stmtErr(&r.Value.Variable.Pos, err)
		}
	}
}

func analyseLimit(u *LimitStmt, c *iterateCtx) {
	err := resolveInCtx(u.RateName, c, func(l *RateStmt, schema *PackageSchemaAST) error { return nil })
	if err != nil {
		c.stmtErr(&u.RateName.Pos, err)
	}
	if u.Action.Tag != nil {
		if err = resolveInCtx(*u.Action.Tag, c, func(t *TagStmt, schema *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&u.Action.Tag.Pos, err)
		}
	} else if u.Action.Command != nil {
		if err = resolveInCtx(*u.Action.Command, c, func(t *CommandStmt, schema *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&u.Action.Command.Pos, err)
		}

	} else if u.Action.Query != nil {
		if err = resolveInCtx(*u.Action.Query, c, func(t *QueryStmt, schema *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&u.Action.Query.Pos, err)
		}
	} else if u.Action.Table != nil {
		if err = resolveInCtx(*u.Action.Table, c, func(t *TableStmt, schema *PackageSchemaAST) error { return nil }); err != nil {
			c.stmtErr(&u.Action.Table.Pos, err)
		}
	}
}

func analyseView(view *ViewStmt, c *iterateCtx) {
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
			rf := fe.RecordField
			if _, ok := fields[string(rf.Name.Value)]; ok {
				c.stmtErr(&rf.Name.Pos, ErrRedefined(string(rf.Name.Value)))
			} else {
				fields[string(rf.Name.Value)] = i
			}
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
	var projector *ProjectorStmt
	err := resolveInCtx(view.ResultOf, c, func(f *ProjectorStmt, _ *PackageSchemaAST) error {
		projector = f
		return nil
	})
	if err != nil {
		c.stmtErr(&view.ResultOf.Pos, err)
		return
	}

	var intentForView *StateStorage
	for i := 0; i < len(projector.Intents) && intentForView == nil; i++ {
		var isView bool
		intent := projector.Intents[i]
		if err := resolveInCtx(intent.Storage, c, func(storage *StorageStmt, _ *PackageSchemaAST) error {
			isView = isView || storage.EntityView
			return nil
		}); err != nil {
			c.stmtErr(&intent.Storage.Pos, err)
		}

		if isView {
			for _, entity := range intent.Entities {
				if entity.Name == view.Name && (entity.Package == Ident(c.pkg.Name) || entity.Package == Ident("")) {
					intentForView = &projector.Intents[i]
					break
				}
			}
		}
	}
	if intentForView == nil {
		c.stmtErr(&view.ResultOf.Pos, ErrProjectorDoesNotDeclareViewIntent(projector.GetName(), view.GetName()))
		return
	}

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
		if !w.Alterable && schema != c.pkg {
			return ErrWorkspaceIsNotAlterable(u.Name.String())
		}
		u.alteredWorkspace = w
		u.alteredWorkspacePkg = schema
		return nil
	}
	err := resolveInCtx(u.Name, c, resolveFunc)
	if err != nil {
		c.stmtErr(&u.Name.Pos, err)
		return
	}
}

func analyseProjector(v *ProjectorStmt, c *iterateCtx) {
	for i := range v.Triggers {
		trigger := &v.Triggers[i]

		if trigger.CronSchedule != nil {
			c.stmtErr(&v.Pos, ErrScheduledProjectorDeprecated)
		}

		for _, qname := range trigger.QNames {
			if len(trigger.TableActions) > 0 {

				wd, pkg, err := lookupInCtx[*WsDescriptorStmt](qname, c)
				if err != nil {
					c.stmtErr(&qname.Pos, err)
					continue
				}
				if wd != nil {
					trigger.qNames = append(trigger.qNames, pkg.NewQName(wd.Name))
					continue
				}

				resolveFunc := func(table *TableStmt, pkg *PackageSchemaAST) error {
					sysDoc := (pkg.Path == appdef.SysPackage) && (table.Name == nameCRecord || table.Name == nameWRecord)
					if table.Abstract && !sysDoc {
						return ErrAbstractTableNotAlowedInProjectors(qname.String())
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
					trigger.qNames = append(trigger.qNames, pkg.NewQName(table.Name))
					return nil
				}
				if err := resolveInCtx(qname, c, resolveFunc); err != nil {
					c.stmtErr(&qname.Pos, err)
				}
			} else { // Command
				if trigger.ExecuteAction.WithParam {
					var pkg *PackageSchemaAST
					var odoc *TableStmt
					typ, pkg, err := lookupInCtx[*TypeStmt](qname, c)
					if err != nil { // type?
						c.stmtErr(&qname.Pos, err)
						continue
					}
					if typ == nil { // ODoc?
						odoc, pkg, err = lookupInCtx[*TableStmt](qname, c)
						if err != nil {
							c.stmtErr(&qname.Pos, err)
							continue
						}
						if odoc == nil || odoc.tableTypeKind != appdef.TypeKind_ODoc {
							c.stmtErr(&qname.Pos, ErrUndefinedTypeOrOdoc(qname))
							continue
						}
					}
					trigger.qNames = append(trigger.qNames, pkg.NewQName(qname.Name))
				} else {
					err := resolveInCtx(qname, c, func(f *CommandStmt, pkg *PackageSchemaAST) error {
						trigger.qNames = append(trigger.qNames, pkg.NewQName(qname.Name))
						return nil
					})
					if err != nil {
						c.stmtErr(&qname.Pos, err)
						continue
					}
				}
			}
		}
	}

	checkState(v.State, c, func(sc *StorageScope) bool { return sc.Projectors })
	checkIntents(v.Intents, c, func(sc *StorageScope) bool { return sc.Projectors })
}

func analyseJob(j *JobStmt, c *iterateCtx) {
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, e := parser.Parse(*j.CronSchedule); e != nil {
		c.stmtErr(&j.Pos, ErrInvalidCronSchedule(*j.CronSchedule))
	}
	checkState(j.State, c, func(sc *StorageScope) bool { return sc.Jobs })
	checkIntents(j.Intents, c, func(sc *StorageScope) bool { return sc.Jobs })
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
			if err := resolveInCtx(tag, c, func(*TagStmt, *PackageSchemaAST) error { return nil }); err != nil {
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

func analyseTable(v *TableStmt, c *iterateCtx) {
	analyseWith(&v.With, v, c)
	analyseNestedTables(v.Items, v.tableTypeKind, c)
	analyseFieldSets(v.Items, c)
	analyseFields(v.Items, c, true)
	if v.Inherits != nil {
		resolvedFunc := func(f *TableStmt, _ *PackageSchemaAST) error {
			if !f.Abstract {
				return ErrBaseTableMustBeAbstract
			}
			return nil
		}
		if err := resolveInCtx(*v.Inherits, c, resolvedFunc); err != nil {
			c.stmtErr(&v.Inherits.Pos, err)
		}

	}
}

func analyseType(v *TypeStmt, c *iterateCtx) {
	for _, i := range v.Items {
		if i.NestedTable != nil {
			c.stmtErr(&i.NestedTable.Pos, ErrNestedTablesNotSupportedInTypes)
		}
	}
	analyseFieldSets(v.Items, c)
	analyseFields(v.Items, c, false)
}

func useStmtInWs(wsctx *wsCtx, stmtPackage *PackageSchemaAST, stmt interface{}) {
	if named, ok := stmt.(INamedStatement); ok {
		if supported(stmt) {
			wsctx.ws.registerNode(stmtPackage.NewQName(Ident(named.GetName())), statementNode{Pkg: stmtPackage, Stmt: named}, wsctx.ws)
		}
	}
	if useTable, ok := stmt.(*UseTableStmt); ok {
		for utQname, ut := range useTable.qNames {
			wsctx.ws.registerNode(utQname, ut, wsctx.ws)
		}
	}
	if useWorkspace, ok := stmt.(*UseWorkspaceStmt); ok {
		if useWorkspace.useWs != nil {
			wsctx.ws.registerNode(useWorkspace.useWs.qName(), *useWorkspace.useWs, wsctx.ws)
		}
	}
}

func analyseWorkspace(v *WorkspaceStmt, c *iterateCtx) {

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
	}

	// find all included QNames
	var iter func(ws *WorkspaceStmt, wsctx *wsCtx, coll IStatementCollection)

	var iterTableItems func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr)
	iterTableItems = func(ws *WorkspaceStmt, wsctx *wsCtx, items []TableItemExpr) {
		for i := range items {
			if items[i].NestedTable != nil {
				useStmtInWs(wsctx, wsctx.pkg, &items[i].NestedTable.Table)
				iterTableItems(ws, wsctx, items[i].NestedTable.Table.Items)
			}
		}
	}

	iter = func(ws *WorkspaceStmt, wsctx *wsCtx, coll IStatementCollection) {
		coll.Iterate(func(stmt interface{}) {
			useStmtInWs(wsctx, wsctx.pkg, stmt)
			if collection, ok := stmt.(IStatementCollection); ok {
				if _, isWorkspace := stmt.(*WorkspaceStmt); !isWorkspace {
					iter(ws, wsctx, collection)
				}
			}
			if t, ok := stmt.(*TableStmt); ok {
				iterTableItems(ws, wsctx, t.Items)
			}
		})
	}

	iter(v, wsc, v)

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
		inheritsAnything := false
		added = append(added, ws)
		for _, inherits := range ws.Inherits {

			inheritsAnything = true
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
			for bws, bwsn := range c.wsCtxs[baseWs].ws.nodes {
				wsctx.ws.registerNode(bws, bwsn.node, c.wsCtxs[baseWs].ws)
			}
			added = append(added, baseWs)
		}
		if !inheritsAnything {
			for sws, swsn := range c.wsCtxs[sysWorkspace].ws.nodes {
				wsctx.ws.registerNode(sws, swsn.node, c.wsCtxs[sysWorkspace].ws)
			}
		}
	}
	addFromInheritedWs(ws, c.wsCtxs[ws])
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
			analyseNestedTables(nestedTable.Items, rootTableKind, c)
		}
	}
}

func analyseFieldSets(items []TableItemExpr, c *iterateCtx) {
	for i := range items {
		item := items[i]
		if item.FieldSet != nil {
			if err := resolveInCtx(item.FieldSet.Type, c, func(*TypeStmt, *PackageSchemaAST) error { return nil }); err != nil {
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
					ws := getCurrentWorkspace(c)
					if ws != nil {
						refQname := tblPkg.NewQName(f.Name)
						if !ws.containsQName(refQname) {
							return ErrReferenceToTableNotInWorkspace(rf.RefDocs[i].String())
						}
					}
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
					ws := getCurrentWorkspace(c)
					if ws != nil {
						refQname := tblPkg.NewQName(f.Name)
						if !ws.containsQName(refQname) {
							return ErrReferenceToTableNotInWorkspace(rf.RefDocs[i].String())
						}
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
			if node.table.Name == nameCDOC {
				kind = appdef.TypeKind_CDoc
			}
			if node.table.Name == nameODOC {
				kind = appdef.TypeKind_ODoc
			}
			if node.table.Name == nameWDOC {
				kind = appdef.TypeKind_WDoc
			}
			if node.table.Name == nameCRecord {
				kind = appdef.TypeKind_CRecord
			}
			if node.table.Name == nameORecord {
				kind = appdef.TypeKind_ORecord
			}
			if node.table.Name == nameWRecord {
				kind = appdef.TypeKind_WRecord
			}
			if node.table.Name == nameCSingleton {
				kind = appdef.TypeKind_CDoc
				singleton = true
			}
			if node.table.Name == nameWSingleton {
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
