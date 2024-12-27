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
	"github.com/voedger/voedger/pkg/appdef/internal/views"
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

func (ws Workspace) Ancestors() iter.Seq[appdef.IWorkspace] {
	return ws.ancestors.Values()
}

func (ws *Workspace) AppendACL(acl appdef.IACLRule) {
	ws.acl = append(ws.acl, acl)
	if app, ok := ws.App().(interface{ AppendACL(appdef.IACLRule) }); ok {
		app.AppendACL(acl) // propagate ACL to app
	}
}

func (ws *Workspace) AppendType(t appdef.IType) {
	if app, ok := ws.App().(interface{ AppendType(appdef.IType) }); ok {
		app.AppendType(t) // propagate type to app
	}
	ws.types.Add(t)
}

func (ws Workspace) Descriptor() appdef.QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return appdef.NullQName
}

func (ws Workspace) Inherits(anc appdef.QName) bool {
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

func (ws Workspace) LocalType(name appdef.QName) appdef.IType {
	return ws.types.Find(name)
}

func (ws Workspace) LocalTypes() iter.Seq[appdef.IType] {
	return ws.types.Values()
}

func (ws Workspace) Type(name appdef.QName) appdef.IType {
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
	return find(&ws)
}

// Enumeration order:
//   - ancestor types recursive from far to nearest
//   - local types
//   - used Workspaces
func (ws Workspace) Types() iter.Seq[appdef.IType] {
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
		visit(&ws)
	}
}

func (ws Workspace) UsedWorkspaces() iter.Seq[appdef.IWorkspace] {
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
	ws.AppendType(t)
	comments.SetComment(&t.WithComments, comment...)
}

func (ws *Workspace) addView(name appdef.QName) appdef.IViewBuilder {
	v := views.NewView(ws, name)
	ws.AppendType(v)
	return views.NewViewBuilder(v)
}

func (ws *Workspace) addWDoc(name appdef.QName) appdef.IWDocBuilder {
	d := structures.NewWDoc(ws, name)
	ws.AppendType(d)
	return structures.NewWDocBuilder(d)
}

func (ws *Workspace) addWRecord(name appdef.QName) appdef.IWRecordBuilder {
	r := structures.NewWRecord(ws, name)
	ws.AppendType(r)
	return structures.NewWRecordBuilder(r)
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

func (ws *Workspace) setAncestors(name appdef.QName, names ...appdef.QName) {
	add := func(n appdef.QName) {
		anc := ws.App().Workspace(n)
		if anc == nil {
			panic(appdef.ErrNotFound("Workspace «%v»", n))
		}
		if anc.Inherits(ws.QName()) {
			panic(appdef.ErrUnsupported("Circular inheritance is not allowed. Workspace «%v» inherits from «%v»", n, ws))
		}
		ws.ancestors.Add(anc)
	}

	ws.ancestors.Clear()
	add(name)
	for _, n := range names {
		add(n)
	}
}

func (ws *Workspace) setDescriptor(q appdef.QName) {
	old := ws.Descriptor()
	if old == q {
		return
	}

	switch q {
	case appdef.NullQName:
		ws.desc = nil
	default:
		d := appdef.CDoc(ws.LocalType, q)
		if d == nil {
			panic(appdef.ErrNotFound("CDoc «%v»", q))
		}
		ws.desc = d
		if d.Abstract() {
			abstracts.SetAbstract(&ws.WithAbstract)
		}
	}

	if app, ok := ws.App().(interface {
		SetWorkspaceDescriptor(appdef.QName, appdef.QName)
	}); ok {
		app.SetWorkspaceDescriptor(ws.QName(), q)
	}
}

func (ws *Workspace) setTypeComment(name appdef.QName, c ...string) {
	t := ws.LocalType(name)
	if t == appdef.NullType {
		panic(appdef.ErrNotFound("type %s", name))
	}
	if t, ok := t.(*types.Typ); ok {
		comments.SetComment(&t.WithComments, c...)
	}
}

func (ws *Workspace) useWorkspace(name appdef.QName, names ...appdef.QName) {
	use := func(n appdef.QName) {
		usedWS := ws.App().Workspace(n)
		if usedWS == nil {
			panic(appdef.ErrNotFound("Workspace «%v»", n))
		}
		ws.usedWS.Add(usedWS)
	}

	use(name)
	for _, n := range names {
		use(n)
	}
}

// # Supports:
//   - appdef.IWorkspaceBuilder
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
	return wb.ws.addCommand(name)
}

func (wb *WorkspaceBuilder) AddCRecord(name appdef.QName) appdef.ICRecordBuilder {
	return wb.ws.addCRecord(name)
}

func (wb *WorkspaceBuilder) AddData(name appdef.QName, kind appdef.DataKind, ancestor appdef.QName, constraints ...appdef.IConstraint) appdef.IDataBuilder {
	return wb.ws.addData(name, kind, ancestor, constraints...)
}

func (wb *WorkspaceBuilder) AddGDoc(name appdef.QName) appdef.IGDocBuilder {
	return wb.ws.addGDoc(name)
}

func (wb *WorkspaceBuilder) AddGRecord(name appdef.QName) appdef.IGRecordBuilder {
	return wb.ws.addGRecord(name)
}

func (wb *WorkspaceBuilder) AddJob(name appdef.QName) appdef.IJobBuilder {
	return wb.ws.addJob(name)
}

func (wb *WorkspaceBuilder) AddLimit(name appdef.QName, ops []appdef.OperationKind, opt appdef.LimitFilterOption, flt appdef.IFilter, rate appdef.QName, comment ...string) {
	wb.ws.addLimit(name, ops, opt, flt, rate, comment...)
}

func (wb *WorkspaceBuilder) AddObject(name appdef.QName) appdef.IObjectBuilder {
	return wb.ws.addObject(name)
}

func (wb *WorkspaceBuilder) AddODoc(name appdef.QName) appdef.IODocBuilder {
	return wb.ws.addODoc(name)
}

func (wb *WorkspaceBuilder) AddORecord(name appdef.QName) appdef.IORecordBuilder {
	return wb.ws.addORecord(name)
}

func (wb *WorkspaceBuilder) AddProjector(name appdef.QName) appdef.IProjectorBuilder {
	return wb.ws.addProjector(name)
}

func (wb *WorkspaceBuilder) AddQuery(name appdef.QName) appdef.IQueryBuilder {
	return wb.ws.addQuery(name)
}

func (wb *WorkspaceBuilder) AddRate(name appdef.QName, count appdef.RateCount, period appdef.RatePeriod, scopes []appdef.RateScope, comment ...string) {
	wb.ws.addRate(name, count, period, scopes, comment...)
}

func (wb *WorkspaceBuilder) AddRole(name appdef.QName) appdef.IRoleBuilder {
	return wb.ws.addRole(name)
}

func (wb *WorkspaceBuilder) AddTag(name appdef.QName, comments ...string) {
	wb.ws.addTag(name, comments...)
}

func (wb *WorkspaceBuilder) AddView(name appdef.QName) appdef.IViewBuilder {
	return wb.ws.addView(name)
}

func (wb *WorkspaceBuilder) AddWDoc(name appdef.QName) appdef.IWDocBuilder {
	return wb.ws.addWDoc(name)
}

func (wb *WorkspaceBuilder) AddWRecord(name appdef.QName) appdef.IWRecordBuilder {
	return wb.ws.addWRecord(name)
}

func (wb *WorkspaceBuilder) Grant(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, toRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.ws.grant(ops, flt, fields, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) GrantAll(flt appdef.IFilter, toRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.ws.grantAll(flt, toRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) Revoke(ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, fromRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.ws.revoke(ops, flt, fields, fromRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) RevokeAll(flt appdef.IFilter, fromRole appdef.QName, comment ...string) appdef.IACLBuilder {
	wb.ws.revokeAll(flt, fromRole, comment...)
	return wb
}

func (wb *WorkspaceBuilder) SetAncestors(name appdef.QName, names ...appdef.QName) appdef.IWorkspaceBuilder {
	wb.ws.setAncestors(name, names...)
	return wb
}

func (wb *WorkspaceBuilder) SetDescriptor(q appdef.QName) appdef.IWorkspaceBuilder {
	wb.ws.setDescriptor(q)
	return wb
}

func (wb *WorkspaceBuilder) SetTypeComment(n appdef.QName, c ...string) {
	wb.ws.setTypeComment(n, c...)
}

func (wb *WorkspaceBuilder) UseWorkspace(name appdef.QName, names ...appdef.QName) appdef.IWorkspaceBuilder {
	wb.ws.useWorkspace(name, names...)
	return wb
}

func (wb *WorkspaceBuilder) Workspace() appdef.IWorkspace { return wb.ws }

// List of Workspaces.
type Workspaces = types.Types[appdef.IWorkspace]

func NewWorkspaces() *Workspaces { return types.NewTypes[appdef.IWorkspace]() }
