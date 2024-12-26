/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package workspaces

import (
	"iter"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/abstracts"
	"github.com/voedger/voedger/pkg/appdef/internal/acl"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
	"github.com/voedger/voedger/pkg/appdef/internal/extensions"
	"github.com/voedger/voedger/pkg/appdef/internal/structures"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IWorkspace
type Workspace struct {
	types.Typ
	abstracts.WithAbstract
	acl       []*acl.Rule
	ancestors *Workspaces
	types     *types.Types[appdef.IType]
	usedWS    *Workspaces
	desc      appdef.ICDoc
}

func NewWorkspace(app appdef.IAppDef, name appdef.QName) *Workspace {
	ws := &Workspace{
		Typ:       types.MakeType(app, nil, name, appdef.TypeKind_Workspace),
		ancestors: NewWorkspaces(),
		types:     types.NewTypes[appdef.IType](),
		usedWS:    NewWorkspaces(),
	}
	if name != appdef.SysWorkspaceQName {
		ws.ancestors.Add(app.Workspace(appdef.SysWorkspaceQName))
	}
	return ws
}

func (ws Workspace) ACL() iter.Seq[appdef.IACLRule] {
	return func(yield func(appdef.IACLRule) bool) {
		for _, acl := range ws.acl {
			if !yield(acl) {
				return
			}
		}
	}
}

func (ws *Workspace) Ancestors() iter.Seq[appdef.IWorkspace] {
	return ws.ancestors.Values()
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
	d := structures.NewCDoc(ws.App(), ws, name)
	ws.AppendType(d)
	return structures.NewCDocBuilder(d)
}

func (ws *Workspace) addCommand(name appdef.QName) appdef.ICommandBuilder {
	c := extensions.NewCommand(ws.App(), ws, name)
	ws.AppendType(c)
	return extensions.NewCommandBuilder(c)
}

func (ws *Workspace) addCRecord(name appdef.QName) appdef.ICRecordBuilder {
	r := structures.NewCRecord(ws.App(), ws, name)
	ws.AppendType(r)
	return structures.NewCRecordBuilder(r)
}

func (ws *Workspace) addData(name appdef.QName, kind appdef.DataKind, ancestor appdef.QName, constraints ...appdef.IConstraint) appdef.IDataBuilder {
	d := datas.NewData(ws.App(), ws, name, kind, ancestor)
	ws.AppendType(d)
	b := datas.NewDataBuilder(d)
	b.AddConstraints(constraints...)
	return b
}

func (ws *Workspace) addGDoc(name appdef.QName) appdef.IGDocBuilder {
	d := structures.NewGDoc(ws.App(), ws, name)
	ws.AppendType(d)
	return structures.NewGDocBuilder(d)
}

func (ws *Workspace) addGRecord(name appdef.QName) appdef.IGRecordBuilder {
	r := structures.NewGRecord(ws.App(), ws, name)
	ws.AppendType(r)
	return structures.NewGRecordBuilder(r)
}

func (ws *Workspace) addJob(name QName) IJobBuilder {
	r := newJob(ws.app, ws, name)
	return newJobBuilder(r)
}

func (ws *Workspace) addLimit(name QName, ops []OperationKind, opt LimitFilterOption, flt IFilter, rate QName, comment ...string) {
	_ = newLimit(ws.app, ws, name, ops, opt, flt, rate, comment...)
}

func (ws *Workspace) addObject(name QName) IObjectBuilder {
	o := newObject(ws.app, ws, name)
	return newObjectBuilder(o)
}

func (ws *Workspace) addODoc(name QName) IODocBuilder {
	d := newODoc(ws.app, ws, name)
	return newODocBuilder(d)
}

func (ws *Workspace) addORecord(name QName) IORecordBuilder {
	r := newORecord(ws.app, ws, name)
	return newORecordBuilder(r)
}

func (ws *Workspace) addProjector(name QName) IProjectorBuilder {
	p := newProjector(ws.app, ws, name)
	return newProjectorBuilder(p)
}

func (ws *Workspace) addQuery(name QName) IQueryBuilder {
	q := newQuery(ws.app, ws, name)
	return newQueryBuilder(q)
}

func (ws *Workspace) addRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	_ = newRate(ws.app, ws, name, count, period, scopes, comment...)
}

func (ws *Workspace) addRole(name QName) IRoleBuilder {
	role := newRole(ws.app, ws, name)
	return newRoleBuilder(role)
}

func (ws *Workspace) addTag(name appdef.QName, comment ...string) {
	t := types.NewTag(ws.App(), ws, name)
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

func (ws *Workspace) AppendACL(p *acl.Rule) {
	ws.acl = append(ws.acl, p)
	ws.App().(interface{ AppendACL(*acl.Rule) }).AppendACL(p)
}

func (ws *Workspace) AppendType(t appdef.IType) {
	ws.App().(interface{ AppendType(appdef.IType) }).AppendType(t)
	// do not check the validity or uniqueness of the name; this was checked by `App().AppendType(t)`
	ws.types.Add(t)
}

func (ws *Workspace) grant(ops []OperationKind, flt IFilter, fields []FieldName, toRole QName, comment ...string) {
	r := appdef.Role(ws.Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	role := r.(*role)

	acl := newGrant(ws, ops, flt, fields, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *Workspace) grantAll(flt IFilter, toRole QName, comment ...string) {
	r := Role(ws.Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	role := r.(*role)

	acl := newGrantAll(ws, flt, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *Workspace) revoke(ops []OperationKind, flt IFilter, fields []FieldName, fromRole QName, comment ...string) {
	r := Role(ws.Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	role := r.(*role)

	acl := newRevoke(ws, ops, flt, fields, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *Workspace) revokeAll(flt IFilter, fromRole QName, comment ...string) {
	r := Role(ws.Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	role := r.(*role)

	acl := newRevokeAll(ws, flt, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
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

func (wb *WorkspaceBuilder) AddJob(name QName) IJobBuilder {
	return wb.Workspace.addJob(name)
}

func (wb *WorkspaceBuilder) AddLimit(name QName, ops []OperationKind, opt LimitFilterOption, flt IFilter, rate QName, comment ...string) {
	wb.Workspace.addLimit(name, ops, opt, flt, rate, comment...)
}

func (wb *WorkspaceBuilder) AddObject(name QName) IObjectBuilder {
	return wb.Workspace.addObject(name)
}

func (wb *WorkspaceBuilder) AddODoc(name QName) IODocBuilder {
	return wb.Workspace.addODoc(name)
}

func (wb *WorkspaceBuilder) AddORecord(name QName) IORecordBuilder {
	return wb.Workspace.addORecord(name)
}

func (wb *WorkspaceBuilder) AddProjector(name QName) IProjectorBuilder {
	return wb.Workspace.addProjector(name)
}

func (wb *WorkspaceBuilder) AddQuery(name QName) IQueryBuilder {
	return wb.Workspace.addQuery(name)
}

func (wb *WorkspaceBuilder) AddRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	wb.Workspace.addRate(name, count, period, scopes, comment...)
}

func (wb *WorkspaceBuilder) AddRole(name QName) IRoleBuilder {
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

func (wb *WorkspaceBuilder) Grant(ops []OperationKind, flt IFilter, fields []FieldName, toRole QName, comment ...string) IACLBuilder {
	wb.Workspace.grant(ops, flt, fields, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) GrantAll(flt IFilter, toRole QName, comment ...string) IACLBuilder {
	wb.Workspace.grantAll(flt, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) Revoke(ops []OperationKind, flt IFilter, fields []FieldName, fromRole QName, comment ...string) IACLBuilder {
	wb.Workspace.revoke(ops, flt, fields, fromRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) RevokeAll(flt IFilter, fromRole QName, comment ...string) IACLBuilder {
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
