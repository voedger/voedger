/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

type analyseCtx struct {
	*basicContext
}

func analyse(c *basicContext, p *PackageSchemaAST) {

	ac := analyseCtx{
		basicContext: c,
	}
	c.pkg = p
	iterate(c.pkg.Ast, func(stmt interface{}) {
		switch v := stmt.(type) {
		case *CommandStmt:
			ac.command(v)
		case *QueryStmt:
			ac.query(v)
		case *ProjectorStmt:
			ac.projector(v)
		case *TableStmt:
			ac.table(v)
		case *WorkspaceStmt:
			ac.workspace(v)
		case *TypeStmt:
			ac.doType(v)
		case *ViewStmt:
			ac.view(v)
		case *UseTableStmt:
			ac.useTable(v)
		case *UseWorkspaceStmt:
			ac.useWorkspace(v)
		case *AlterWorkspaceStmt:
			ac.alterWorkspace(v)
		}
	})
}

func (c *analyseCtx) useTable(u *UseTableStmt) {
	tbl, err := resolveTable(DefQName{Package: c.pkg.Ast.Package, Name: u.Table}, c.basicContext)
	if err != nil {
		c.stmtErr(&u.Pos, err)
	} else {
		if tbl.Abstract {
			c.stmtErr(&u.Pos, ErrUseOfAbstractTable(string(u.Table)))
		}
		// TODO: Only documents allowed to be USEd, not records
	}
}

func (c *analyseCtx) useWorkspace(u *UseWorkspaceStmt) {
	resolveFunc := func(f *WorkspaceStmt) error {
		if f.Abstract {
			return ErrUseOfAbstractWorkspace(string(u.Workspace))
		}
		return nil
	}
	err := resolve[*WorkspaceStmt](DefQName{Package: c.pkg.Ast.Package, Name: u.Workspace}, c.basicContext, resolveFunc)
	if err != nil {
		c.stmtErr(&u.Pos, err)
	}
}

func (c *analyseCtx) alterWorkspace(u *AlterWorkspaceStmt) {
	resolveFunc := func(w *WorkspaceStmt, schema *PackageSchemaAST) error {
		if !w.Alterable && schema != c.pkg {
			return ErrWorkspaceIsNotAlterable(u.Name.String())
		}
		return nil
	}
	err := resolveEx[*WorkspaceStmt](u.Name, c.basicContext, resolveFunc)
	if err != nil {
		c.stmtErr(&u.Pos, err)
	}
}

func (c *analyseCtx) view(view *ViewStmt) {
	view.pkRef = nil
	fields := make(map[string]int)
	for i := range view.Fields {
		fe := view.Fields[i]
		if fe.PrimaryKey != nil {
			if view.pkRef != nil {
				c.stmtErr(&fe.Pos, ErrPrimaryKeyRedeclared)
			} else {
				view.pkRef = fe.PrimaryKey
			}
		}
		if fe.Field != nil {
			if _, ok := fields[string(fe.Field.Name)]; ok {
				c.stmtErr(&fe.Pos, ErrRedeclared(string(fe.Field.Name)))
			} else {
				fields[string(fe.Field.Name)] = i
			}
		}
	}
	if view.pkRef == nil {
		c.stmtErr(&view.Pos, ErrPrimaryKeyNotDeclared)
	}

}

func (c *analyseCtx) command(v *CommandStmt) {
	if v.Arg != nil && !isVoid(v.Arg.Package, v.Arg.Name) {
		if getDefDataKind(v.Arg.Package, v.Arg.Name) == appdef.DataKind_null {
			if err := resolve(*v.Arg, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else {
			c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForArgument)
		}
	}
	if v.UnloggedArg != nil && !isVoid(v.UnloggedArg.Package, v.UnloggedArg.Name) {
		if getDefDataKind(v.UnloggedArg.Package, v.UnloggedArg.Name) == appdef.DataKind_null {
			if err := resolve(*v.UnloggedArg, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else {
			c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForArgument)
		}
	}
	if v.Returns != nil && !isVoid(v.Returns.Package, v.Returns.Name) {
		if getDefDataKind(v.Returns.Package, v.Returns.Name) == appdef.DataKind_null {
			if err := resolve(*v.Returns, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else {
			c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForResult)
		}
	}
	c.with(&v.With, v)
}

func (c *analyseCtx) query(v *QueryStmt) {
	if v.Arg != nil && !isVoid(v.Arg.Package, v.Arg.Name) {
		if getDefDataKind(v.Arg.Package, v.Arg.Name) == appdef.DataKind_null {
			if err := resolve(*v.Arg, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else {
			c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForArgument)
		}
	}
	if !isAny(v.Returns.Package, v.Returns.Name) {
		if !isVoid(v.Returns.Package, v.Returns.Name) {
			if getDefDataKind(v.Returns.Package, v.Returns.Name) == appdef.DataKind_null {
				if err := resolve(v.Returns, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
					c.stmtErr(&v.Pos, err)
				}
			} else {
				c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForResult)
			}
		}

	}
	c.with(&v.With, v)

}
func (c *analyseCtx) projector(v *ProjectorStmt) {
	for _, target := range v.Triggers {
		if v.On.Activate || v.On.Deactivate || v.On.Insert || v.On.Update {
			resolveFunc := func(f *TableStmt) error {
				if f.Abstract {
					return ErrAbstractTableNotAlowedInProjectors(target.String())
				}
				return nil
			}
			if err := resolve(target, c.basicContext, resolveFunc); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else if v.On.Command {
			if err := resolve(target, c.basicContext, func(f *CommandStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else if v.On.CommandArgument {
			if err := resolve(target, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		}
	}

	checkEntity := func(key StorageKey, f *StorageStmt) error {
		if f.EntityRecord {
			if key.Entity == nil {
				return ErrStorageRequiresEntity(key.Storage.String())
			}
			resolveFunc := func(f *TableStmt) error {
				if f.Abstract {
					return ErrAbstractTableNotAlowedInProjectors(key.Entity.String())
				}
				return nil
			}
			if err2 := resolve(*key.Entity, c.basicContext, resolveFunc); err2 != nil {
				return err2
			}
		}
		if f.EntityView {
			if key.Entity == nil {
				return ErrStorageRequiresEntity(key.Storage.String())
			}
			if err2 := resolve(*key.Entity, c.basicContext, func(f *ViewStmt) error { return nil }); err2 != nil {
				return err2
			}
		}
		return nil
	}

	for _, key := range v.State {
		if err := resolve(key.Storage, c.basicContext, func(f *StorageStmt) error {
			if e := checkEntity(key, f); e != nil {
				return e
			}
			read := false
			for _, op := range f.Ops {
				if op.Get || op.GetBatch || op.Read {
					for _, sc := range op.Scope {
						if sc.Projectors {
							read = true
							break
						}
					}
				}
			}
			if !read {
				return ErrStorageNotInProjectorState(key.Storage.String())
			}
			return nil
		}); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}

	for _, key := range v.Intents {
		if err := resolve(key.Storage, c.basicContext, func(f *StorageStmt) error {
			if e := checkEntity(key, f); e != nil {
				return e
			}
			read := false
			for _, op := range f.Ops {
				if op.Insert || op.Update {
					for _, sc := range op.Scope {
						if sc.Projectors {
							read = true
							break
						}
					}
				}
			}
			if !read {
				return ErrStorageNotInProjectorIntents(key.Storage.String())
			}
			return nil
		}); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}

}

// Note: function may update with argument
func (c *analyseCtx) with(with *[]WithItem, statement IStatement) {
	var comment *WithItem

	for i := range *with {
		item := &(*with)[i]
		if item.Comment != nil {
			comment = item
		} else if item.Rate != nil {
			if err := resolve(*item.Rate, c.basicContext, func(f *RateStmt) error { return nil }); err != nil {
				c.stmtErr(statement.GetPos(), err)
			}
		}
		for j := range item.Tags {
			tag := item.Tags[j]
			if err := resolve(tag, c.basicContext, func(f *TagStmt) error { return nil }); err != nil {
				c.stmtErr(statement.GetPos(), err)
			}
		}
	}

	if comment != nil {
		statement.SetComments(strings.Split(*comment.Comment, "\n"))
	}
}

func (c *analyseCtx) table(v *TableStmt) {
	if isPredefinedSysTable(c.pkg.QualifiedPackageName, v) {
		return
	}
	var err error
	v.tableDefKind, v.singletone, err = c.getTableDefKind(v)
	if err != nil {
		c.stmtErr(&v.Pos, err)
		return
	}
	c.with(&v.With, v)
	c.nestedTables(v.Items, v.tableDefKind)
	c.fieldSets(v.Items)
	c.fields(v.Items)
	if v.Inherits != nil {
		resolvedFunc := func(f *TableStmt) error {
			if !f.Abstract {
				return ErrBaseTableMustBeAbstract
			}
			return nil
		}
		if err := resolve(*v.Inherits, c.basicContext, resolvedFunc); err != nil {
			c.stmtErr(&v.Pos, err)
		}

	}
}

func (c *analyseCtx) doType(v *TypeStmt) {
	for _, i := range v.Items {
		if i.NestedTable != nil {
			c.stmtErr(&v.Pos, ErrNestedTablesNotSupportedInTypes)
		}
	}
	c.fieldSets(v.Items)
}

func (c *analyseCtx) workspace(v *WorkspaceStmt) {
	for _, inherits := range v.Inherits {
		resolveFunc := func(w *WorkspaceStmt) error {
			if !w.Abstract {
				return ErrBaseWorkspaceMustBeAbstract
			}
			return nil
		}
		if err := resolve(inherits, c.basicContext, resolveFunc); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	if v.Descriptor != nil {
		if v.Abstract {
			c.stmtErr(&v.Pos, ErrAbstractWorkspaceDescriptor)
		}
		c.nestedTables(v.Descriptor.Items, appdef.DefKind_CDoc)
		c.fieldSets(v.Descriptor.Items)
	}
}

func (c *analyseCtx) nestedTables(items []TableItemExpr, rootTableKind appdef.DefKind) {
	for i := range items {
		item := items[i]
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			if nestedTable.Abstract {
				c.stmtErr(&nestedTable.Pos, ErrNestedAbstractTable(nestedTable.GetName()))
				return
			}
			if nestedTable.Inherits == nil {
				nestedTable.tableDefKind = getNestedTableKind(rootTableKind)
			} else {
				var err error
				nestedTable.tableDefKind, nestedTable.singletone, err = c.getTableDefKind(nestedTable)
				if err != nil {
					c.stmtErr(&nestedTable.Pos, err)
					return
				}
				tk := getNestedTableKind(rootTableKind)
				if nestedTable.tableDefKind != tk {
					c.stmtErr(&nestedTable.Pos, ErrNestedTableIncorrectKind)
					return
				}
			}
			c.nestedTables(nestedTable.Items, rootTableKind)
		}
	}
}

func (c *analyseCtx) fieldSets(items []TableItemExpr) {
	for i := range items {
		item := items[i]
		if item.FieldSet != nil {
			if err := resolve(item.FieldSet.Type, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&item.FieldSet.Pos, err)
				continue
			}
		}
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			c.fieldSets(nestedTable.Items)
		}
	}
}

func (c *analyseCtx) fields(items []TableItemExpr) {
	for i := range items {
		item := items[i]
		if item.RefField != nil {
			rf := item.RefField
			for i := range rf.RefDocs {
				tableStmt, err := resolveTable(rf.RefDocs[i], c.basicContext)
				if err != nil {
					c.stmtErr(&rf.Pos, err)
					continue
				}
				if tableStmt.Abstract {
					c.stmtErr(&rf.Pos, ErrReferenceToAbstractTable(rf.RefDocs[i].String()))
					continue
				}
			}
		}
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
			c.fields(nestedTable.Items)
		}
	}
}

func (c *analyseCtx) getTableInheritanceChain(table *TableStmt) (chain []DefQName, err error) {
	chain = make([]DefQName, 0)
	var vf func(t *TableStmt) error
	vf = func(t *TableStmt) error {
		if t.Inherits != nil {
			inherited := *t.Inherits
			if err := resolve(inherited, c.basicContext, func(t *TableStmt) error {
				chain = append(chain, inherited)
				return vf(t)
			}); err != nil {
				return err
			}
		}
		return nil
	}
	err = vf(table)
	return
}

func (c *analyseCtx) getTableDefKind(table *TableStmt) (kind appdef.DefKind, singletone bool, err error) {
	chain, e := c.getTableInheritanceChain(table)
	if e != nil {
		return appdef.DefKind_null, false, e
	}
	for _, t := range chain {
		if isSysDef(t, nameCDOC) || isSysDef(t, nameSingleton) {
			return appdef.DefKind_CDoc, isSysDef(t, nameSingleton), nil
		} else if isSysDef(t, nameODOC) {
			return appdef.DefKind_ODoc, false, nil
		} else if isSysDef(t, nameWDOC) {
			return appdef.DefKind_WDoc, false, nil
		} else if isSysDef(t, nameCRecord) {
			return appdef.DefKind_CRecord, false, nil
		} else if isSysDef(t, nameORecord) {
			return appdef.DefKind_ORecord, false, nil
		} else if isSysDef(t, nameWRecord) {
			return appdef.DefKind_WRecord, false, nil
		}
	}
	return appdef.DefKind_null, false, ErrUndefinedTableKind
}
