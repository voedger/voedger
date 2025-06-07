/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package workspaces

import (
	"errors"
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
	acl.WithACL
	types.WithTypes
	allTypes  []appdef.IType
	ancestors *Workspaces
	usedWS    *Workspaces
	desc      appdef.ICDoc
}

func NewWorkspace(app appdef.IAppDef, name appdef.QName) *Workspace {
	ws := &Workspace{
		Typ:          types.MakeType(app, nil, name, appdef.TypeKind_Workspace),
		WithAbstract: abstracts.MakeWithAbstract(),
		WithACL:      acl.MakeWithACL(),
		WithTypes:    types.MakeWithTypes(),
		ancestors:    NewWorkspaces(),
		usedWS:       NewWorkspaces(),
	}
	if name != appdef.SysWorkspaceQName {
		ws.ancestors.Add(app.Workspace(appdef.SysWorkspaceQName))
	}
	types.Propagate(ws)
	return ws
}

func (ws Workspace) Ancestors() []appdef.IWorkspace {
	return ws.ancestors.AsArray()
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
	return ws.WithTypes.Type(name)
}

func (ws Workspace) LocalTypes() []appdef.IType {
	return ws.WithTypes.Types()
}

func (ws Workspace) Type(name appdef.QName) appdef.IType {
	var (
		find  func(appdef.IWorkspace) appdef.IType
		chain = make(map[appdef.QName]bool) // to prevent stack overflow recursion
	)
	find = func(w appdef.IWorkspace) appdef.IType {
		if !chain[w.QName()] {
			chain[w.QName()] = true
			if t := w.LocalType(name); t != appdef.NullType {
				return t
			}
			for _, a := range w.Ancestors() {
				if t := find(a); t != appdef.NullType {
					return t
				}
			}
			for _, u := range w.UsedWorkspaces() {
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

func (ws Workspace) Types() []appdef.IType {
	if ws.allTypes != nil {
		return ws.allTypes
	}
	return ws.enumerateTypes()
}

func (ws Workspace) UsedWorkspaces() []appdef.IWorkspace {
	return ws.usedWS.AsArray()
}

func (ws *Workspace) Validate() (err error) {
	for _, t := range ws.LocalTypes() {
		if t, ok := t.(interface{ Validate() error }); ok {
			err = errors.Join(err, t.Validate())
		}
	}

	err = errors.Join(err, ws.ValidateACL())

	if (ws.desc != nil) && ws.desc.Abstract() && !ws.Abstract() {
		err = errors.Join(err, appdef.ErrIncompatible("%v should be abstract because descriptor %v is abstract", ws, ws.desc))
	}
	return err
}

func (ws *Workspace) addCDoc(name appdef.QName) appdef.ICDocBuilder {
	d := structures.NewCDoc(ws, name)
	return structures.NewCDocBuilder(d)
}

func (ws *Workspace) addCommand(name appdef.QName) appdef.ICommandBuilder {
	c := extensions.NewCommand(ws, name)
	return extensions.NewCommandBuilder(c)
}

func (ws *Workspace) addCRecord(name appdef.QName) appdef.ICRecordBuilder {
	r := structures.NewCRecord(ws, name)
	return structures.NewCRecordBuilder(r)
}

func (ws *Workspace) addData(name appdef.QName, kind appdef.DataKind, ancestor appdef.QName, constraints ...appdef.IConstraint) appdef.IDataBuilder {
	d := datas.NewData(ws, name, kind, ancestor)
	b := datas.NewDataBuilder(d)
	b.AddConstraints(constraints...)
	return b
}

func (ws *Workspace) addGDoc(name appdef.QName) appdef.IGDocBuilder {
	d := structures.NewGDoc(ws, name)
	return structures.NewGDocBuilder(d)
}

func (ws *Workspace) addGRecord(name appdef.QName) appdef.IGRecordBuilder {
	r := structures.NewGRecord(ws, name)
	return structures.NewGRecordBuilder(r)
}

func (ws *Workspace) addJob(name appdef.QName) appdef.IJobBuilder {
	j := extensions.NewJob(ws, name)
	return extensions.NewJobBuilder(j)
}

func (ws *Workspace) addLimit(name appdef.QName, ops []appdef.OperationKind, opt appdef.LimitFilterOption, flt appdef.IFilter, rate appdef.QName, comment ...string) {
	rates.NewLimit(ws, name, ops, opt, flt, rate, comment...)
}

func (ws *Workspace) addObject(name appdef.QName) appdef.IObjectBuilder {
	o := structures.NewObject(ws, name)
	return structures.NewObjectBuilder(o)
}

func (ws *Workspace) addODoc(name appdef.QName) appdef.IODocBuilder {
	d := structures.NewODoc(ws, name)
	return structures.NewODocBuilder(d)
}

func (ws *Workspace) addORecord(name appdef.QName) appdef.IORecordBuilder {
	r := structures.NewORecord(ws, name)
	return structures.NewORecordBuilder(r)
}

func (ws *Workspace) addProjector(name appdef.QName) appdef.IProjectorBuilder {
	p := extensions.NewProjector(ws, name)
	return extensions.NewProjectorBuilder(p)
}

func (ws *Workspace) addQuery(name appdef.QName) appdef.IQueryBuilder {
	q := extensions.NewQuery(ws, name)
	return extensions.NewQueryBuilder(q)
}

func (ws *Workspace) addRate(name appdef.QName, count appdef.RateCount, period appdef.RatePeriod, scopes []appdef.RateScope, comment ...string) {
	rates.NewRate(ws, name, count, period, scopes, comment...)
}

func (ws *Workspace) addRole(name appdef.QName) appdef.IRoleBuilder {
	r := roles.NewRole(ws, name)
	return roles.NewRoleBuilder(r)
}

func (ws *Workspace) addTag(name appdef.QName, featureAndComment ...string) {
	feature := ""
	if len(featureAndComment) > 0 {
		feature = featureAndComment[0]
	}
	t := types.NewTag(ws, name, feature)
	if len(featureAndComment) > 1 {
		comments.SetComment(&t.WithComments, featureAndComment[1:]...)
	}
}

func (ws *Workspace) addView(name appdef.QName) appdef.IViewBuilder {
	v := views.NewView(ws, name)
	return views.NewViewBuilder(v)
}

func (ws *Workspace) addWDoc(name appdef.QName) appdef.IWDocBuilder {
	d := structures.NewWDoc(ws, name)
	return structures.NewWDocBuilder(d)
}

func (ws *Workspace) addWRecord(name appdef.QName) appdef.IWRecordBuilder {
	r := structures.NewWRecord(ws, name)
	return structures.NewWRecordBuilder(r)
}

func (ws *Workspace) build() error {
	return ws.Validate()
}

// Should be called after ws successfully built.
func (ws *Workspace) builded() {
	ws.allTypes = ws.enumerateTypes()
}

func (ws *Workspace) changed() { ws.allTypes = nil }

func (ws Workspace) enumerateTypes() []appdef.IType {
	tt := []appdef.IType{}

	var (
		visit func(appdef.IWorkspace)
		chain = make(map[appdef.QName]bool) // to prevent stack overflow recursion
	)
	visit = func(w appdef.IWorkspace) {
		if !chain[w.QName()] {
			chain[w.QName()] = true
			for _, a := range w.Ancestors() {
				visit(a)
			}
			tt = append(tt, w.LocalTypes()...)
			for _, u := range w.UsedWorkspaces() {
				// #2872 should enum used Workspaces, but not types from them
				tt = append(tt, u)
			}
		}
	}
	visit(&ws)

	slices.SortFunc(tt, func(i, j appdef.IType) int { return appdef.CompareQName(i.QName(), j.QName()) })

	return tt
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

func (wb *WorkspaceBuilder) AddTag(name appdef.QName, featureAndComment ...string) {
	wb.ws.addTag(name, featureAndComment...)
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

func (wb *WorkspaceBuilder) UseWorkspace(name appdef.QName, names ...appdef.QName) appdef.IWorkspaceBuilder {
	wb.ws.useWorkspace(name, names...)
	return wb
}

func (wb *WorkspaceBuilder) Workspace() appdef.IWorkspace { return wb.ws }

// List of Workspaces.
type Workspaces = types.Types[appdef.IWorkspace]

func NewWorkspaces() *Workspaces { return types.NewTypes[appdef.IWorkspace]() }
