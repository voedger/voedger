/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IWorkspace
type workspace struct {
	typ
	withAbstract
	acl              []*aclRule
	ancestors        map[QName]IWorkspace
	ancestorsOrdered QNames
	types            struct {
		local, all *types
	}
	usedWS        map[QName]IWorkspace
	usedWSOrdered QNames
	desc          ICDoc
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		typ:              makeType(app, nil, name, TypeKind_Workspace),
		ancestors:        make(map[QName]IWorkspace),
		ancestorsOrdered: QNames{},
		usedWS:           make(map[QName]IWorkspace),
		usedWSOrdered:    QNames{},
	}
	ws.types.local = newTypes()

	if name != SysWorkspaceQName {
		ws.ancestors[SysWorkspaceQName] = app.Workspace(SysWorkspaceQName)
		ws.ancestorsOrdered.Add(SysWorkspaceQName)
	}

	app.appendType(ws)
	return ws
}

func (ws workspace) ACL(cb func(IACLRule) bool) {
	for _, p := range ws.acl {
		if !cb(p) {
			break
		}
	}
}

func (ws *workspace) Ancestors(recurse bool) []QName {
	res := QNamesFrom(ws.ancestorsOrdered...)
	if recurse {
		for _, a := range ws.ancestors {
			res.Add(a.Ancestors(true)...)
		}
	}
	return res
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
		for _, a := range ws.ancestors {
			if a.Inherits(anc) {
				return true
			}
		}
	}
	return false
}

func (ws *workspace) LocalType(name QName) IType {
	return ws.types.local.Type(name)
}

func (ws *workspace) LocalTypes(visit func(IType) bool) {
	ws.types.local.Types(visit)
}

func (ws *workspace) Type(name QName) IType {
	return ws.allTypes().Type(name)
}

func (ws *workspace) Types(visit func(IType) bool) {
	ws.allTypes().Types(visit)
}

func (ws *workspace) UsedWorkspaces() []QName {
	return QNamesFrom(ws.usedWSOrdered...)
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

func (ws *workspace) addLimit(name QName, on []QName, rate QName, comment ...string) {
	_ = newLimit(ws.app, ws, name, on, rate, comment...)
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

func (ws *workspace) appendType(t interface{}) {
	ws.app.appendType(t)

	// do not check the validity or uniqueness of the name; this was checked by `*application.appendType (t)`

	ws.types.local.append(t)
	ws.types.all = nil
}

// returns list of all types, include types from ancestor workspaces and used workspaces recursively
func (ws *workspace) allTypes() *types {
	if ws.types.all == nil {
		ws.types.all = newTypes()

		var (
			append func(*workspace)
			chain  map[QName]bool = make(map[QName]bool) // to prevent stack overflow recursion
		)

		append = func(w *workspace) {
			if chain[w.QName()] {
				return
			}
			chain[w.QName()] = true
			ws.types.all.append(w)
			for t := range w.types.local.Types {
				ws.types.all.append(t)
			}
			for _, a := range w.ancestors {
				append(a.(*workspace))
			}
			for _, u := range w.usedWS {
				append(u.(*workspace))
			}
		}

		append(ws)
	}
	return ws.types.all
}

func (ws *workspace) grant(ops []OperationKind, resources []QName, fields []FieldName, toRole QName, comment ...string) {
	r := Role(ws.allTypes().Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	r.(*role).grant(ops, resources, fields, comment...)
}

func (ws *workspace) grantAll(resources []QName, toRole QName, comment ...string) {
	r := Role(ws.allTypes().Type, toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	r.(*role).grantAll(resources, comment...)
}

func (ws *workspace) revoke(ops []OperationKind, resources []QName, fields []FieldName, fromRole QName, comment ...string) {
	r := Role(ws.allTypes().Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	r.(*role).revoke(ops, resources, fields, comment...)
}

func (ws *workspace) revokeAll(resources []QName, fromRole QName, comment ...string) {
	r := Role(ws.allTypes().Type, fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	r.(*role).revokeAll(resources, comment...)
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
		ws.ancestors[n] = anc
		ws.ancestorsOrdered.Add(n)
	}

	clear(ws.ancestors)
	ws.ancestorsOrdered = QNames{}

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

	if ws.desc = CDoc(ws.types.local.Type, q); ws.desc == nil {
		panic(ErrNotFound("CDoc «%v»", q))
	}
	if ws.desc.Abstract() {
		ws.withAbstract.setAbstract()
	}

	ws.app.wsDesc[q] = ws
}

func (ws *workspace) useWorkspace(name QName, names ...QName) {
	use := func(n QName) {
		usedWS := ws.app.Workspace(n)
		if usedWS == nil {
			panic(ErrNotFound("Workspace «%v»", n))
		}
		if _, ok := ws.usedWS[n]; ok {
			panic(ErrAlreadyExists("%v already used by %v", usedWS, ws))
		}
		ws.usedWS[n] = usedWS
		ws.usedWSOrdered.Add(n)
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

func (wb *workspaceBuilder) AddLimit(name QName, on []QName, rate QName, comment ...string) {
	wb.workspace.addLimit(name, on, rate, comment...)
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

func (wb *workspaceBuilder) AddRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	wb.workspace.addRate(name, count, period, scopes, comment...)
}

func (wb *workspaceBuilder) AddRole(name QName) IRoleBuilder {
	return wb.workspace.addRole(name)
}

func (wb *workspaceBuilder) AddQuery(name QName) IQueryBuilder {
	return wb.workspace.addQuery(name)
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

func (wb *workspaceBuilder) Grant(ops []OperationKind, resources []QName, fields []FieldName, toRole QName, comment ...string) IACLBuilder {
	wb.workspace.grant(ops, resources, fields, toRole, comment...)
	return wb
}

func (wb *workspaceBuilder) GrantAll(resources []QName, toRole QName, comment ...string) IACLBuilder {
	wb.workspace.grantAll(resources, toRole, comment...)
	return wb
}

func (wb *workspaceBuilder) Revoke(ops []OperationKind, resources []QName, fields []FieldName, fromRole QName, comment ...string) IACLBuilder {
	wb.workspace.revoke(ops, resources, fields, fromRole, comment...)
	return wb
}

func (wb *workspaceBuilder) RevokeAll(resources []QName, fromRole QName, comment ...string) IACLBuilder {
	wb.workspace.revokeAll(resources, fromRole, comment...)
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

func (wb *workspaceBuilder) UseWorkspace(name QName, names ...QName) IWorkspaceBuilder {
	wb.workspace.useWorkspace(name, names...)
	return wb
}

func (wb *workspaceBuilder) Workspace() IWorkspace { return wb.workspace }
