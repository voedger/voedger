/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package workspaces

import (
	"iter"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/abstracts"
	"github.com/voedger/voedger/pkg/appdef/internal/acl"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
	"github.com/voedger/voedger/pkg/appdef/internal/extensions"
	"github.com/voedger/voedger/pkg/appdef/internal/rates"
	"github.com/voedger/voedger/pkg/appdef/internal/roles"
	"github.com/voedger/voedger/pkg/appdef/internal/structures"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IWorkspace
type Workspace struct {
	types.Typ
	abstracts.WithAbstract
	acl       []appdef.IACLRule
	ancestors *Workspaces
	types     *types.Types[appdef.IType]
	usedWS    *Workspaces
	desc      appdef.ICDoc
}

func NewWorkspace(app appdef.IAppDef, name appdef.QName) *Workspace {
	ws := &Workspace{
		Typ:          types.MakeType(app, nil, name, appdef.TypeKind_Workspace),
		WithAbstract: abstracts.MakeWithAbstract(),
		acl:          make([]appdef.IACLRule, 0),
		ancestors:    NewWorkspaces(),
		types:        types.NewTypes[appdef.IType](),
		usedWS:       NewWorkspaces(),
	}
	if name != appdef.SysWorkspaceQName {
		ws.ancestors.Add(app.Workspace(appdef.SysWorkspaceQName))
	}
	return ws
}

func (ws Workspace) ACL() iter.Seq[appdef.IACLRule] { return slices.Values(ws.acl) }

func (ws *Workspace) Ancestors() iter.Seq[appdef.IWorkspace] {
	return ws.ancestors.Values()
}

func (ws *Workspace) AppendACL(acl appdef.IACLRule) {
	ws.acl = append(ws.acl, acl)
	if app, ok := ws.App().(interface{ AppendACL(appdef.IACLRule) }); ok {
		app.AppendACL(acl) // propagate ACL to app
	}
}

func (ws *Workspace) AppendType(t appdef.IType) {
	ws.App().(interface{ AppendType(appdef.IType) }).AppendType(t)
	// do not check the validity or uniqueness of the name; this was checked by `App().AppendType(t)`
	ws.types.Add(t)
}

func (ws *Workspace) Descriptor() appdef.QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return appdef.NullQName
}

func (ws *Workspace) Inherits(anc appdef.QName) bool {
	switch anc {
	case appdef.SysWorkspaceQName, ws.QName():
		return true
	default:
		for a := range ws.ancestors.Values() {
			if a.Inherits(anc) {
				return true
			}
		}
	}
	return false
}

func (ws *Workspace) LocalType(name appdef.QName) appdef.IType {
	return ws.types.Find(name)
}

func (ws *Workspace) LocalTypes() iter.Seq[appdef.IType] {
	return ws.types.Values()
}

func (ws *Workspace) Type(name appdef.QName) appdef.IType {
	var (
		find  func(appdef.IWorkspace) appdef.IType
		chain map[appdef.QName]bool = make(map[appdef.QName]bool) // to prevent stack overflow recursion
	)
	find = func(w appdef.IWorkspace) appdef.IType {
		if !chain[w.QName()] {
			chain[w.QName()] = true
			if t := w.LocalType(name); t != appdef.NullType {
				return t
			}
			for a := range w.Ancestors() {
				if t := find(a); t != appdef.NullType {
					return t
				}
			}
			for u := range w.UsedWorkspaces() {
				// #2872 should find used Workspaces, but not types from them
				if u.QName() == name {
					return u
				}
			}
		}
		return appdef.NullType
	}
	return find(ws)
}

// Enumeration order:
//   - ancestor types recursive from far to nearest
//   - local types
//   - used Workspaces
func (ws *Workspace) Types() iter.Seq[appdef.IType] {
	return func(yield func(appdef.IType) bool) {
		var (
			visit func(appdef.IWorkspace) bool
			chain map[appdef.QName]bool = make(map[appdef.QName]bool) // to prevent stack overflow recursion
		)
		visit = func(w appdef.IWorkspace) bool {
			if !chain[w.QName()] {
				chain[w.QName()] = true
				for a := range w.Ancestors() {
					if !visit(a) {
						return false
					}
				}
				for t := range w.LocalTypes() {
					if !yield(t) {
						return false
					}
				}
				for u := range w.UsedWorkspaces() {
					// #2872 should enum used Workspaces, but not types from them
					if !yield(u) {
						return false
					}
				}
			}
			return true
		}
		visit(ws)
	}
}

func (ws *Workspace) UsedWorkspaces() iter.Seq[appdef.IWorkspace] {
	return ws.usedWS.Values()
}

func (ws *Workspace) Validate() error {
	if (ws.desc != nil) && ws.desc.Abstract() && !ws.Abstract() {
		return appdef.ErrIncompatible("%v should be abstract because descriptor %v is abstract", ws, ws.desc)
	}
	return nil
}

func (ws *Workspace) addCDoc(name appdef.QName) appdef.ICDocBuilder {
	d := structures.NewCDoc(ws, name)
	ws.AppendType(d)
	return structures.NewCDocBuilder(d)
}

func (ws *Workspace) addCommand(name appdef.QName) appdef.ICommandBuilder {
	c := extensions.NewCommand(ws, name)
	ws.AppendType(c)
	return extensions.NewCommandBuilder(c)
}

func (ws *Workspace) addCRecord(name appdef.QName) appdef.ICRecordBuilder {
	r := structures.NewCRecord(ws, name)
	ws.AppendType(r)
	return structures.NewCRecordBuilder(r)
}

func (ws *Workspace) addData(name appdef.QName, kind appdef.DataKind, ancestor appdef.QName, constraints ...appdef.IConstraint) appdef.IDataBuilder {
	d := datas.NewData(ws, name, kind, ancestor)
	ws.AppendType(d)
	b := datas.NewDataBuilder(d)
	b.AddConstraints(constraints...)
	return b
}

func (ws *Workspace) addGDoc(name appdef.QName) appdef.IGDocBuilder {
	d := structures.NewGDoc(ws, name)
	ws.AppendType(d)
	return structures.NewGDocBuilder(d)
}

func (ws *Workspace) addGRecord(name appdef.QName) appdef.IGRecordBuilder {
	r := structures.NewGRecord(ws, name)
	ws.AppendType(r)
	return structures.NewGRecordBuilder(r)
}

func (ws *Workspace) addJob(name appdef.QName) appdef.IJobBuilder {
	j := extensions.NewJob(ws, name)
	ws.AppendType(j)
	return extensions.NewJobBuilder(j)
}

func (ws *Workspace) addLimit(name appdef.QName, ops []appdef.OperationKind, opt appdef.LimitFilterOption, flt appdef.IFilter, rate appdef.QName, comment ...string) {
	l := rates.NewLimit(ws, name, ops, opt, flt, rate, comment...)
	ws.AppendType(l)
}

func (ws *Workspace) addObject(name appdef.QName) appdef.IObjectBuilder {
	o := structures.NewObject(ws, name)
	ws.AppendType(o)
	return structures.NewObjectBuilder(o)
}

func (ws *Workspace) addODoc(name appdef.QName) appdef.IODocBuilder {
	d := structures.NewODoc(ws, name)
	ws.AppendType(d)
	return structures.NewODocBuilder(d)
}

func (ws *Workspace) addORecord(name appdef.QName) appdef.IORecordBuilder {
	r := structures.NewORecord(ws, name)
	ws.AppendType(r)
	return structures.NewORecordBuilder(r)
}

func (ws *Workspace) addProjector(name appdef.QName) appdef.IProjectorBuilder {
	p := extensions.NewProjector(ws, name)
	ws.AppendType(p)
	return extensions.NewProjectorBuilder(p)
}

func (ws *Workspace) addQuery(name appdef.QName) appdef.IQueryBuilder {
	q := extensions.NewQuery(ws, name)
	ws.AppendType(q)
	return extensions.NewQueryBuilder(q)
}

func (ws *Workspace) addRate(name appdef.QName, count appdef.RateCount, period appdef.RatePeriod, scopes []appdef.RateScope, comment ...string) {
	r := rates.NewRate(ws, name, count, period, scopes, comment...)
	ws.AppendType(r)
}

func (ws *Workspace) addRole(name appdef.QName) appdef.IRoleBuilder {
	r := roles.NewRole(ws, name)
	ws.AppendType(r)
	return roles.NewRoleBuilder(r)
}

func (ws *Workspace) addTag(name appdef.QName, comment ...string) {
	t := types.NewTag(ws, name)
	comments.SetComment(t.WithComments, comment...)
}

func (ws *Workspace) addView(name QName) IViewBuilder {
	v := newView(ws.app, ws, name)
	return newViewBuilder(v)
}

func (ws *Workspace) addWDoc(name QName) IWDocBuilder {
	d := newWDoc(ws.app, ws, name)
	return newWDocBuilder(d)
}

func (ws *Workspace) addWRecord(name QName) IWRecordBuilder {
	r := newWRecord(ws.app, ws, name)
	return newWRecordBuilder(r)
}

func (ws *Workspace) grant(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, toRole appdef.QName, comment ...string) {
	r := appdef.Role(ws.Type, toRole)
	if r == nil {
		panic(appdef.ErrRoleNotFound(toRole))
	}
	acl.NewGrant(ws, ops, flt, fields, r, comment...)
}

func (ws *Workspace) grantAll(flt appdef.IFilter, toRole appdef.QName, comment ...string) {
	r := appdef.Role(ws.Type, toRole)
	if r == nil {
		panic(appdef.ErrRoleNotFound(toRole))
	}
	acl.NewGrantAll(ws, flt, r, comment...)
}

func (ws *Workspace) revoke(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, fromRole appdef.QName, comment ...string) {
	r := appdef.Role(ws.Type, fromRole)
	if r == nil {
		panic(appdef.ErrRoleNotFound(fromRole))
	}
	acl.NewRevoke(ws, ops, flt, fields, r, comment...)
}

func (ws *Workspace) revokeAll(flt appdef.IFilter, fromRole appdef.QName, comment ...string) {
	r := appdef.Role(ws.Type, fromRole)
	if r == nil {
		panic(appdef.ErrRoleNotFound(fromRole))
	}
	acl.NewRevokeAll(ws, flt, r, comment...)
}

func (ws *Workspace) setAncestors(name QName, names ...QName) {
	add := func(n QName) {
		anc := ws.app.Workspace(n)
		if anc == nil {
			panic(ErrNotFound("Workspace «%v»", n))
		}
		if anc.Inherits(ws.QName()) {
			panic(ErrUnsupported("Circular inheritance is not allowed. Workspace «%v» inherits from «%v»", n, ws))
		}
		ws.ancestors.add(anc)
	}

	ws.ancestors.clear()
	add(name)
	for _, n := range names {
		add(n)
	}
}

func (ws *Workspace) setDescriptor(q QName) {
	old := ws.Descriptor()
	if old == q {
		return
	}

	if (old != NullQName) && (ws.app.wsDesc[old] == ws) {
		delete(ws.app.wsDesc, old)
	}

	if q == NullQName {
		ws.desc = nil
		return
	}

	if ws.desc = CDoc(ws.LocalType, q); ws.desc == nil {
		panic(ErrNotFound("CDoc «%v»", q))
	}
	if ws.desc.Abstract() {
		ws.withAbstract.setAbstract()
	}

	ws.app.wsDesc[q] = ws
}

func (ws *Workspace) setTypeComment(name QName, c ...string) {
	t := ws.LocalType(name)
	if t == NullType {
		panic(ErrNotFound("type %s", name))
	}
	if t, ok := t.(interface{ setComment(...string) }); ok {
		t.setComment(c...)
	}
}

func (ws *Workspace) useWorkspace(name QName, names ...QName) {
	use := func(n QName) {
		usedWS := ws.app.Workspace(n)
		if usedWS == nil {
			panic(ErrNotFound("Workspace «%v»", n))
		}
		ws.usedWS.add(usedWS)
	}

	use(name)
	for _, n := range names {
		use(n)
	}
}

// # Implements:
//   - IWorkspaceBuilder
type WorkspaceBuilder struct {
	types.TypeBuilder
	abstracts.WithAbstractBuilder
	ws *Workspace
}

func NewWorkspaceBuilder(ws *Workspace) *WorkspaceBuilder {
	return &WorkspaceBuilder{
		TypeBuilder:         types.MakeTypeBuilder(&ws.Typ),
		WithAbstractBuilder: abstracts.MakeWithAbstractBuilder(&ws.WithAbstract),
		ws:                  ws,
	}
}

func (wb *WorkspaceBuilder) AddCDoc(name appdef.QName) appdef.ICDocBuilder {
	return wb.ws.addCDoc(name)
}

func (wb *WorkspaceBuilder) AddCommand(name appdef.QName) appdef.ICommandBuilder {
	return wb.Workspace.addCommand(name)
}

func (wb *WorkspaceBuilder) AddCRecord(name appdef.QName) appdef.ICRecordBuilder {
	return wb.Workspace.addCRecord(name)
}

func (wb *WorkspaceBuilder) AddData(name appdef.QName, kind appdef.DataKind, ancestor appdef.QName, constraints ...appdef.IConstraint) appdef.IDataBuilder {
	return wb.ws.addData(name, kind, ancestor, constraints...)
}

func (wb *WorkspaceBuilder) AddGDoc(name appdef.QName) appdef.IGDocBuilder {
	return wb.Workspace.addGDoc(name)
}

func (wb *WorkspaceBuilder) AddGRecord(name appdef.QName) appdef.IGRecordBuilder {
	return wb.Workspace.addGRecord(name)
}

func (wb *WorkspaceBuilder) AddJob(name appdef.QName) appdef.IJobBuilder {
	return wb.Workspace.addJob(name)
}

func (wb *WorkspaceBuilder) AddLimit(name appdef.QName, ops []appdef.OperationKind, opt appdef.LimitFilterOption, flt appdef.IFilter, rate appdef.QName, comment ...string) {
	wb.Workspace.addLimit(name, ops, opt, flt, rate, comment...)
}

func (wb *WorkspaceBuilder) AddObject(name appdef.QName) appdef.IObjectBuilder {
	return wb.Workspace.addObject(name)
}

func (wb *WorkspaceBuilder) AddODoc(name appdef.QName) appdef.IODocBuilder {
	return wb.Workspace.addODoc(name)
}

func (wb *WorkspaceBuilder) AddORecord(name appdef.QName) appdef.IORecordBuilder {
	return wb.Workspace.addORecord(name)
}

func (wb *WorkspaceBuilder) AddProjector(name appdef.QName) appdef.IProjectorBuilder {
	return wb.Workspace.addProjector(name)
}

func (wb *WorkspaceBuilder) AddQuery(name appdef.QName) appdef.IQueryBuilder {
	return wb.Workspace.addQuery(name)
}

func (wb *WorkspaceBuilder) AddRate(name appdef.QName, count appdef.RateCount, period appdef.RatePeriod, scopes []appdef.RateScope, comment ...string) {
	wb.Workspace.addRate(name, count, period, scopes, comment...)
}

func (wb *WorkspaceBuilder) AddRole(name appdef.QName) appdef.IRoleBuilder {
	return wb.Workspace.addRole(name)
}

func (wb *WorkspaceBuilder) AddTag(name QName, comments ...string) {
	wb.Workspace.addTag(name, comments...)
}

func (wb *WorkspaceBuilder) AddView(name QName) IViewBuilder {
	return wb.Workspace.addView(name)
}

func (wb *WorkspaceBuilder) AddWDoc(name QName) IWDocBuilder {
	return wb.Workspace.addWDoc(name)
}

func (wb *WorkspaceBuilder) AddWRecord(name QName) IWRecordBuilder {
	return wb.Workspace.addWRecord(name)
}

func (wb *WorkspaceBuilder) Grant(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, toRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.Workspace.grant(ops, flt, fields, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) GrantAll(flt appdef.IFilter, toRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.Workspace.grantAll(flt, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) Revoke(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, fromRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.Workspace.revoke(ops, flt, fields, fromRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) RevokeAll(flt appdef.IFilter, fromRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.Workspace.revokeAll(flt, fromRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) SetAncestors(name QName, names ...QName) IWorkspaceBuilder {
	wb.Workspace.setAncestors(name, names...)
	return wb
}

func (wb *WorkspaceBuilder) SetDescriptor(q QName) IWorkspaceBuilder {
	wb.Workspace.setDescriptor(q)
	return wb
}

func (wb *WorkspaceBuilder) SetTypeComment(n QName, c ...string) {
	wb.Workspace.setTypeComment(n, c...)
}

func (wb *WorkspaceBuilder) UseWorkspace(name QName, names ...QName) IWorkspaceBuilder {
	wb.Workspace.useWorkspace(name, names...)
	return wb
}

func (wb *WorkspaceBuilder) Workspace() IWorkspace { return wb.workspace }

// List of Workspaces.
type Workspaces = types.Types[appdef.IWorkspace]

func NewWorkspaces() *Workspaces { return types.NewTypes[appdef.IWorkspace]() }
