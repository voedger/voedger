/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"github.com/alecthomas/participle/v2/lexer"
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
		}
	})
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
			if _, ok := fields[fe.Field.Name]; ok {
				c.stmtErr(&fe.Pos, ErrRedeclared(fe.Field.Name))
			} else {
				fields[fe.Field.Name] = i
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
	c.with(v.With, &v.Pos)
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
	if !isVoid(v.Returns.Package, v.Returns.Name) {
		if getDefDataKind(v.Returns.Package, v.Returns.Name) == appdef.DataKind_null {
			if err := resolve(v.Returns, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else {
			c.stmtErr(&v.Pos, ErrOnlyTypeOrVoidAllowedForResult)
		}
	}
	c.with(v.With, &v.Pos)

}
func (c *analyseCtx) projector(v *ProjectorStmt) {
	for _, target := range v.Triggers {
		if v.On.Activate || v.On.Deactivate || v.On.Insert || v.On.Update {
			if err := resolve(target, c.basicContext, func(f *TableStmt) error { return nil }); err != nil {
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
			if err2 := resolve(*key.Entity, c.basicContext, func(f *TableStmt) error { return nil }); err2 != nil {
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

func (c *analyseCtx) with(with []WithItem, pos *lexer.Position) {
	for i := range with {
		wi := &with[i]
		if wi.Comment != nil {
			if err := resolve(*wi.Comment, c.basicContext, func(f *CommentStmt) error { return nil }); err != nil {
				c.stmtErr(pos, err)
			}
		} else if wi.Rate != nil {
			if err := resolve(*wi.Rate, c.basicContext, func(f *RateStmt) error { return nil }); err != nil {
				c.stmtErr(pos, err)
			}
		}
		for j := range wi.Tags {
			tag := wi.Tags[j]
			if err := resolve(tag, c.basicContext, func(f *TagStmt) error { return nil }); err != nil {
				c.stmtErr(pos, err)
			}
		}
	}
}

func (c *analyseCtx) table(v *TableStmt) {
	if isPredefinedSysTable(c.pkg.QualifiedPackageName, v) {
		return
	}
	var err error
	v.tableDefKind, v.singletone, err = c.getTableDefKind(v)
	if err != nil {
		c.stmtErr(&v.Pos, nil)
		return
	}
	c.with(v.With, &v.Pos)
	c.nestedTables(v.Items, v.tableDefKind)
	if v.Inherits != nil {
		if err := resolve(*v.Inherits, c.basicContext, func(f *TableStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	for _, of := range v.Of {
		if err := resolve(of, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
}

func (c *analyseCtx) doType(v *TypeStmt) {
	for _, of := range v.Of {
		if err := resolve(of, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	for _, i := range v.Items {
		if i.NestedTable != nil {
			c.stmtErr(&v.Pos, ErrNestedTablesNotSupportedInTypes)
		}
	}
}

func (c *analyseCtx) workspace(v *WorkspaceStmt) {
	if v.Descriptor != nil {
		for _, of := range v.Of {
			if err := resolve(of, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		}
		for _, of := range v.Of {
			if err := resolve(of, c.basicContext, func(f *WorkspaceStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		}
		for _, of := range v.Descriptor.Of {
			if err := resolve(of, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		}
		c.nestedTables(v.Descriptor.Items, appdef.DefKind_CDoc)
	}
}

func (c *analyseCtx) nestedTables(items []TableItemExpr, rootTableKind appdef.DefKind) {
	for i := range items {
		item := items[i]
		if item.NestedTable != nil {
			nestedTable := &item.NestedTable.Table
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
