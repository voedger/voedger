/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"regexp"
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
	tbl, _, err := resolveTable(DefQName{Package: c.pkg.Ast.Package, Name: u.Table}, c.basicContext)
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
	err := resolve(DefQName{Package: c.pkg.Ast.Package, Name: u.Workspace}, c.basicContext, resolveFunc)
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
	err := resolveEx(u.Name, c.basicContext, resolveFunc)
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
				c.stmtErr(&fe.PrimaryKey.Pos, ErrPrimaryKeyRedeclared)
			} else {
				view.pkRef = fe.PrimaryKey
			}
		}
		if fe.Field != nil {
			f := fe.Field
			if _, ok := fields[string(f.Name)]; ok {
				c.stmtErr(&f.Pos, ErrRedeclared(string(f.Name)))
			} else {
				fields[string(f.Name)] = i
			}
		}
		if fe.RefField != nil {
			rf := fe.RefField
			for i := range rf.RefDocs {
				tableStmt, _, err := resolveTable(rf.RefDocs[i], c.basicContext)
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
	}
	if view.pkRef == nil {
		c.stmtErr(&view.Pos, ErrPrimaryKeyNotDeclared)
	}
	for _, pkf := range view.pkRef.PartitionKeyFields {
		index, ok := fields[string(pkf)]
		if !ok {
			c.stmtErr(&view.pkRef.Pos, ErrUndefinedField(string(pkf)))
		}
		if view.Fields[index].Field.Type.Varchar != nil {
			c.stmtErr(&view.pkRef.Pos, ErrViewFieldVarchar(string(pkf)))
		}
		if view.Fields[index].Field.Type.Bytes != nil {
			c.stmtErr(&view.pkRef.Pos, ErrViewFieldBytes(string(pkf)))
		}
	}

	for ccIndex, ccf := range view.pkRef.ClusteringColumnsFields {
		fieldIndex, ok := fields[string(ccf)]
		last := ccIndex == len(view.pkRef.ClusteringColumnsFields)-1
		if !ok {
			c.stmtErr(&view.pkRef.Pos, ErrUndefinedField(string(ccf)))
		}
		if view.Fields[fieldIndex].Field.Type.Varchar != nil && !last {
			c.stmtErr(&view.pkRef.Pos, ErrVarcharFieldInCC(string(ccf)))
		}
		if view.Fields[fieldIndex].Field.Type.Bytes != nil && !last {
			c.stmtErr(&view.pkRef.Pos, ErrBytesFieldInCC(string(ccf)))
		}
	}

}

func (c *analyseCtx) command(v *CommandStmt) {
	if v.Arg != nil && v.Arg.Def != nil {
		if err := resolve(*v.Arg.Def, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	if v.UnloggedArg != nil && v.UnloggedArg.Def != nil {
		if err := resolve(*v.UnloggedArg.Def, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	if v.Returns != nil && v.Returns.Def != nil {
		if err := resolve(*v.Returns.Def, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	c.with(&v.With, v)
}

func (c *analyseCtx) query(v *QueryStmt) {
	if v.Arg != nil && v.Arg.Def != nil {
		if err := resolve(*v.Arg.Def, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}

	}
	if v.Returns.Def != nil {
		if err := resolve(*v.Returns.Def, c.basicContext, func(f *TypeStmt) error { return nil }); err != nil {
			c.stmtErr(&v.Pos, err)
		}
	}
	c.with(&v.With, v)

}
func (c *analyseCtx) projector(v *ProjectorStmt) {
	for _, target := range v.On {
		if v.CUDEvents != nil {
			resolveFunc := func(table *TableStmt) error {
				if table.Abstract {
					return ErrAbstractTableNotAlowedInProjectors(target.String())
				}
				defKind, _, err := c.getTableDefKind(table)
				if err != nil {
					return err
				}
				if defKind == appdef.DefKind_ODoc || defKind == appdef.DefKind_ORecord {
					if v.CUDEvents.Activate || v.CUDEvents.Deactivate || v.CUDEvents.Update {
						return ErrOnlyInsertForOdocOrORecord
					}
				}
				return nil
			}
			if err := resolve(target, c.basicContext, resolveFunc); err != nil {
				c.stmtErr(&v.Pos, err)
			}
		} else { // The type of ON not defined
			// Command?
			cmd, _, err := lookup[*CommandStmt](target, c.basicContext)
			if err != nil {
				c.stmtErr(&v.Pos, err)
				continue
			}
			if cmd != nil {
				continue // resolved
			}

			// Command Argument?
			cmdArg, _, err := lookup[*TypeStmt](target, c.basicContext)
			if err != nil {
				c.stmtErr(&v.Pos, err)
				continue
			}
			if cmdArg != nil {
				continue // resolved
			}

			// Table?
			table, _, err := lookup[*TableStmt](target, c.basicContext)
			if err != nil {
				c.stmtErr(&v.Pos, err)
				continue
			}
			if table == nil {
				c.stmtErr(&v.Pos, ErrUndefinedExpectedCommandTypeOrTable(target))
				continue
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
	c.fields(v.Items)
}

func (c *analyseCtx) workspace(v *WorkspaceStmt) {

	var chain []DefQName
	var checkChain func(qn DefQName) error

	checkChain = func(qn DefQName) error {
		resolveFunc := func(w *WorkspaceStmt) error {
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
			return nil
		}
		return resolve(qn, c.basicContext, resolveFunc)
	}

	for _, inherits := range v.Inherits {
		chain = make([]DefQName, 0)
		if err := checkChain(inherits); err != nil {
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
		if item.Field != nil {
			field := item.Field
			if field.CheckRegexp != nil {
				if field.Type.DataType != nil && field.Type.DataType.Varchar != nil {
					_, err := regexp.Compile(*field.CheckRegexp)
					if err != nil {
						c.stmtErr(&field.Pos, ErrCheckRegexpErr(err))
					}
				} else {
					c.stmtErr(&field.Pos, ErrRegexpCheckOnlyForVarcharField)
				}
			}
			if field.Type.DataType != nil && field.Type.DataType.Varchar != nil && field.Type.DataType.Varchar.MaxLen != nil {
				if *field.Type.DataType.Varchar.MaxLen > appdef.MaxFieldLength {
					c.stmtErr(&field.Pos, ErrMaxFieldLengthTooLarge)
				}
			}
		}
		if item.RefField != nil {
			rf := item.RefField
			for i := range rf.RefDocs {
				tableStmt, _, err := resolveTable(rf.RefDocs[i], c.basicContext)
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
	refCycle := func(qname DefQName) bool {
		for i := range chain {
			if chain[i] == qname {
				return true
			}
		}
		return false
	}
	var vf func(t *TableStmt) error
	vf = func(t *TableStmt) error {
		if t.Inherits != nil {
			inherited := *t.Inherits
			if err := resolve(inherited, c.basicContext, func(t *TableStmt) error {
				if refCycle(inherited) {
					return ErrCircularReferenceInInherits
				}
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
