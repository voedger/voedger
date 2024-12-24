/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"iter"
)

// # Implements:
//   - IWorkspace
type workspace struct {
	typ
	withAbstract
	acl       []*aclRule
	ancestors *workspaces
	types     *types[IType]
	usedWS    *workspaces
	desc      ICDoc
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		typ:       makeType(app, nil, name, TypeKind_Workspace),
		ancestors: newWorkspaces(),
		types:     newTypes[IType](),
		usedWS:    newWorkspaces(),
	}
	if name != SysWorkspaceQName {
		ws.ancestors.add(app.Workspace(SysWorkspaceQName))
	}

	app.appendType(ws)
	app.appendWorkspace(ws)
	return ws
}

func (ws workspace) ACL() iter.Seq[IACLRule] {
	return func(yield func(IACLRule) bool) {
		for _, acl := range ws.acl {
			if !yield(acl) {
				return
			}
		}
	}
}

func (ws *workspace) Ancestors() iter.Seq[IWorkspace] {
	return ws.ancestors.values()
}

func (ws *workspace) Descriptor() QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return NullQName
}

func (ws *workspace) Inherits(anc QName) bool {
	switch anc {
	case SysWorkspaceQName, ws.QName():
		return true
	default:
		for a := range ws.ancestors.values() {
			if a.Inherits(anc) {
				return true
			}
		}
	}
	return false
}

func (ws *workspace) LocalType(name QName) IType {
	return ws.types.find(name)
}

func (ws *workspace) LocalTypes() iter.Seq[IType] {
	return ws.types.values()
}

func (ws *workspace) Type(name QName) IType {
	var (
		find  func(IWorkspace) IType
		chain map[QName]bool = make(map[QName]bool) // to prevent stack overflow recursion
	)
	find = func(w IWorkspace) IType {
		if !chain[w.QName()] {
			chain[w.QName()] = true
			if t := w.LocalType(name); t != NullType {
				return t
			}
			for a := range w.Ancestors() {
				if t := find(a.(*workspace)); t != NullType {
					return t
				}
			}
			for u := range w.UsedWorkspaces() {
				// #2872 should find used workspaces, but not types from them
				if u.QName() == name {
					return u
				}
			}
		}
		return NullType
	}
	return find(ws)
}

// Enumeration order:
//   - ancestor types recursive from far to nearest
//   - local types
//   - used workspaces
func (ws *workspace) Types() iter.Seq[IType] {
	return func(yield func(IType) bool) {
		var (
			visit func(IWorkspace) bool
			chain map[QName]bool = make(map[QName]bool) // to prevent stack overflow recursion
		)
		visit = func(w IWorkspace) bool {
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
					// #2872 should enum used workspaces, but not types from them
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

func (ws *workspace) UsedWorkspaces() iter.Seq[IWorkspace] {
	return ws.usedWS.values()
}

func (ws *workspace) Validate() error {
	if (ws.desc != nil) && ws.desc.Abstract() && !ws.Abstract() {
		return ErrIncompatible("%v should be abstract because descriptor %v is abstract", ws, ws.desc)
	}
	return nil
}

func (ws *workspace) addCDoc(name QName) ICDocBuilder {
	d := newCDoc(ws.app, ws, name)
	return newCDocBuilder(d)
}

func (ws *workspace) addCommand(name QName) ICommandBuilder {
	c := newCommand(ws.app, ws, name)
	return newCommandBuilder(c)
}

func (ws *workspace) addCRecord(name QName) ICRecordBuilder {
	r := newCRecord(ws.app, ws, name)
	return newCRecordBuilder(r)
}

func (ws *workspace) addData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	d := newData(ws.app, ws, name, kind, ancestor)
	d.addConstraints(constraints...)
	ws.appendType(d)
	return newDataBuilder(d)
}

func (ws *workspace) addGDoc(name QName) IGDocBuilder {
	d := newGDoc(ws.app, ws, name)
	return newGDocBuilder(d)
}

func (ws *workspace) addGRecord(name QName) IGRecordBuilder {
	r := newGRecord(ws.app, ws, name)
	return newGRecordBuilder(r)
}

func (ws *workspace) addJob(name QName) IJobBuilder {
	r := newJob(ws.app, ws, name)
	return newJobBuilder(r)
}

func (ws *workspace) addLimit(name QName, ops []OperationKind, opt LimitFilterOption, flt IFilter, rate QName, comment ...string) {
	_ = newLimit(ws.app, ws, name, ops, opt, flt, rate, comment...)
}

func (ws *workspace) addObject(name QName) IObjectBuilder {
	o := newObject(ws.app, ws, name)
	return newObjectBuilder(o)
}

func (ws *workspace) addODoc(name QName) IODocBuilder {
	d := newODoc(ws.app, ws, name)
	return newODocBuilder(d)
}

func (ws *workspace) addORecord(name QName) IORecordBuilder {
	r := newORecord(ws.app, ws, name)
	return newORecordBuilder(r)
}

func (ws *workspace) addProjector(name QName) IProjectorBuilder {
	p := newProjector(ws.app, ws, name)
	return newProjectorBuilder(p)
}

func (ws *workspace) addQuery(name QName) IQueryBuilder {
	q := newQuery(ws.app, ws, name)
	return newQueryBuilder(q)
}

func (ws *workspace) addRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	_ = newRate(ws.app, ws, name, count, period, scopes, comment...)
}

func (ws *workspace) addRole(name QName) IRoleBuilder {
	role := newRole(ws.app, ws, name)
	return newRoleBuilder(role)
}

func (ws *workspace) addTag(name QName, comments ...string) {
	t := newTag(ws.app, ws, name)
	t.setComment(comments...)
}

func (ws *workspace) addView(name QName) IViewBuilder {
	v := newView(ws.app, ws, name)
	return newViewBuilder(v)
}

func (ws *workspace) addWDoc(name QName) IWDocBuilder {
	d := newWDoc(ws.app, ws, name)
	return newWDocBuilder(d)
}

func (ws *workspace) addWRecord(name QName) IWRecordBuilder {
	r := newWRecord(ws.app, ws, name)
	return newWRecordBuilder(r)
}

func (ws *workspace) appendACL(p *aclRule) {
	ws.acl = append(ws.acl, p)
	ws.app.appendACL(p)
}

func (ws *workspace) appendType(t IType) {
	ws.app.appendType(t)
	// do not check the validity or uniqueness of the name; this was checked by `*application.appendType (t)`
	ws.types.add(t)
}

func (ws *workspace) grant(ops []OperationKind, flt IFilter, fields []FieldName, toRole QName, comment ...string) {
	r := Role(ws.Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	role := r.(*role)

	acl := newGrant(ws, ops, flt, fields, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *workspace) grantAll(flt IFilter, toRole QName, comment ...string) {
	r := Role(ws.Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	role := r.(*role)

	acl := newGrantAll(ws, flt, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *workspace) revoke(ops []OperationKind, flt IFilter, fields []FieldName, fromRole QName, comment ...string) {
	r := Role(ws.Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	role := r.(*role)

	acl := newRevoke(ws, ops, flt, fields, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *workspace) revokeAll(flt IFilter, fromRole QName, comment ...string) {
	r := Role(ws.Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	role := r.(*role)

	acl := newRevokeAll(ws, flt, role, comment...)
	role.appendACL(acl)
	ws.appendACL(acl)
}

func (ws *workspace) setAncestors(name QName, names ...QName) {
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

func (ws *workspace) setDescriptor(q QName) {
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

func (ws *workspace) setTypeComment(name QName, c ...string) {
	t := ws.LocalType(name)
	if t == NullType {
		panic(ErrNotFound("type %s", name))
	}
	if t, ok := t.(interface{ setComment(...string) }); ok {
		t.setComment(c...)
	}
}

func (ws *workspace) useWorkspace(name QName, names ...QName) {
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
type workspaceBuilder struct {
	typeBuilder
	withAbstractBuilder
	*workspace
}

func newWorkspaceBuilder(workspace *workspace) *workspaceBuilder {
	return &workspaceBuilder{
		typeBuilder:         makeTypeBuilder(&workspace.typ),
		withAbstractBuilder: makeWithAbstractBuilder(&workspace.withAbstract),
		workspace:           workspace,
	}
}

func (wb *workspaceBuilder) AddData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	return wb.workspace.addData(name, kind, ancestor, constraints...)
}

func (wb *workspaceBuilder) AddCDoc(name QName) ICDocBuilder {
	return wb.workspace.addCDoc(name)
}

func (wb *workspaceBuilder) AddCommand(name QName) ICommandBuilder {
	return wb.workspace.addCommand(name)
}

func (wb *workspaceBuilder) AddCRecord(name QName) ICRecordBuilder {
	return wb.workspace.addCRecord(name)
}

func (wb *workspaceBuilder) AddGDoc(name QName) IGDocBuilder {
	return wb.workspace.addGDoc(name)
}

func (wb *workspaceBuilder) AddGRecord(name QName) IGRecordBuilder {
	return wb.workspace.addGRecord(name)
}

func (wb *workspaceBuilder) AddJob(name QName) IJobBuilder {
	return wb.workspace.addJob(name)
}

func (wb *workspaceBuilder) AddLimit(name QName, ops []OperationKind, opt LimitFilterOption, flt IFilter, rate QName, comment ...string) {
	wb.workspace.addLimit(name, ops, opt, flt, rate, comment...)
}

func (wb *workspaceBuilder) AddObject(name QName) IObjectBuilder {
	return wb.workspace.addObject(name)
}

func (wb *workspaceBuilder) AddODoc(name QName) IODocBuilder {
	return wb.workspace.addODoc(name)
}

func (wb *workspaceBuilder) AddORecord(name QName) IORecordBuilder {
	return wb.workspace.addORecord(name)
}

func (wb *workspaceBuilder) AddProjector(name QName) IProjectorBuilder {
	return wb.workspace.addProjector(name)
}

func (wb *workspaceBuilder) AddQuery(name QName) IQueryBuilder {
	return wb.workspace.addQuery(name)
}

func (wb *workspaceBuilder) AddRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	wb.workspace.addRate(name, count, period, scopes, comment...)
}

func (wb *workspaceBuilder) AddRole(name QName) IRoleBuilder {
	return wb.workspace.addRole(name)
}

func (wb *workspaceBuilder) AddTag(name QName, comments ...string) {
	wb.workspace.addTag(name, comments...)
}

func (wb *workspaceBuilder) AddView(name QName) IViewBuilder {
	return wb.workspace.addView(name)
}

func (wb *workspaceBuilder) AddWDoc(name QName) IWDocBuilder {
	return wb.workspace.addWDoc(name)
}

func (wb *workspaceBuilder) AddWRecord(name QName) IWRecordBuilder {
	return wb.workspace.addWRecord(name)
}

func (wb *workspaceBuilder) Grant(ops []OperationKind, flt IFilter, fields []FieldName, toRole QName, comment ...string) IACLBuilder {
	wb.workspace.grant(ops, flt, fields, toRole, comment...)
	return wb
}

func (wb *workspaceBuilder) GrantAll(flt IFilter, toRole QName, comment ...string) IACLBuilder {
	wb.workspace.grantAll(flt, toRole, comment...)
	return wb
}

func (wb *workspaceBuilder) Revoke(ops []OperationKind, flt IFilter, fields []FieldName, fromRole QName, comment ...string) IACLBuilder {
	wb.workspace.revoke(ops, flt, fields, fromRole, comment...)
	return wb
}

func (wb *workspaceBuilder) RevokeAll(flt IFilter, fromRole QName, comment ...string) IACLBuilder {
	wb.workspace.revokeAll(flt, fromRole, comment...)
	return wb
}

func (wb *workspaceBuilder) SetAncestors(name QName, names ...QName) IWorkspaceBuilder {
	wb.workspace.setAncestors(name, names...)
	return wb
}

func (wb *workspaceBuilder) SetDescriptor(q QName) IWorkspaceBuilder {
	wb.workspace.setDescriptor(q)
	return wb
}

func (wb *workspaceBuilder) SetTypeComment(n QName, c ...string) {
	wb.workspace.setTypeComment(n, c...)
}

func (wb *workspaceBuilder) UseWorkspace(name QName, names ...QName) IWorkspaceBuilder {
	wb.workspace.useWorkspace(name, names...)
	return wb
}

func (wb *workspaceBuilder) Workspace() IWorkspace { return wb.workspace }

// List of workspaces.
type workspaces = types[IWorkspace]

func newWorkspaces() *workspaces { return newTypes[IWorkspace]() }
