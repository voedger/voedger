/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"sort"
)

// # Implements:
//   - IWorkspace
type workspace struct {
	typ
	withAbstract
	ancestors        map[QName]IWorkspace
	ancestorsOrdered QNames
	types            map[QName]interface{}
	typesOrdered     []interface{}
	desc             ICDoc
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		typ:       makeType(app, nil, name, TypeKind_Workspace),
		ancestors: make(map[QName]IWorkspace),
		types:     make(map[QName]interface{}),
	}

	if name != SysWorkspaceQName {
		ws.ancestors[SysWorkspaceQName] = app.Workspace(SysWorkspaceQName)
	}

	app.appendType(ws)
	return ws
}

func (ws *workspace) Ancestors() []QName {
	if len(ws.ancestorsOrdered) != len(ws.ancestors) {
		ws.ancestorsOrdered = QNamesFromMap(ws.ancestors)
	}
	return ws.ancestorsOrdered
}

func (ws *workspace) CDoc(name QName) ICDoc {
	if t := ws.typeByKind(name, TypeKind_CDoc); t != nil {
		return t.(ICDoc)
	}
	return nil
}

func (ws *workspace) CDocs(cb func(ICDoc) bool) {
	for t := range ws.Types {
		if d, ok := t.(ICDoc); ok {
			if !cb(d) {
				break
			}
		}
	}
}

func (ws *workspace) CRecord(name QName) ICRecord {
	if t := ws.typeByKind(name, TypeKind_CRecord); t != nil {
		return t.(ICRecord)
	}
	return nil
}

func (ws *workspace) CRecords(cb func(ICRecord) bool) {
	for t := range ws.Types {
		if r, ok := t.(ICRecord); ok {
			if !cb(r) {
				break
			}
		}
	}
}

func (ws *workspace) Descriptor() QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return NullQName
}

func (ws *workspace) Data(name QName) IData {
	if t := ws.typeByKind(name, TypeKind_Data); t != nil {
		return t.(IData)
	}
	return nil
}

func (ws *workspace) DataTypes(cb func(IData) bool) {
	for t := range ws.Types {
		if d, ok := t.(IData); ok {
			if !cb(d) {
				break
			}
		}
	}
}

func (ws *workspace) GDoc(name QName) IGDoc {
	if t := ws.typeByKind(name, TypeKind_GDoc); t != nil {
		return t.(IGDoc)
	}
	return nil
}

func (ws *workspace) GDocs(cb func(IGDoc) bool) {
	for t := range ws.Types {
		if d, ok := t.(IGDoc); ok {
			if !cb(d) {
				break
			}
		}
	}
}

func (ws *workspace) GRecord(name QName) IGRecord {
	if t := ws.typeByKind(name, TypeKind_GRecord); t != nil {
		return t.(IGRecord)
	}
	return nil
}

func (ws *workspace) GRecords(cb func(IGRecord) bool) {
	for t := range ws.Types {
		if r, ok := t.(IGRecord); ok {
			if !cb(r) {
				break
			}
		}
	}
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

func (ws *workspace) Object(name QName) IObject {
	if t := ws.typeByKind(name, TypeKind_Object); t != nil {
		return t.(IObject)
	}
	return nil
}

func (ws *workspace) Objects(cb func(IObject) bool) {
	for t := range ws.Types {
		if o, ok := t.(IObject); ok {
			if !cb(o) {
				break
			}
		}
	}
}

func (ws *workspace) ODoc(name QName) IODoc {
	if t := ws.typeByKind(name, TypeKind_ODoc); t != nil {
		return t.(IODoc)
	}
	return nil
}

func (ws *workspace) ODocs(cb func(IODoc) bool) {
	for t := range ws.Types {
		if d, ok := t.(IODoc); ok {
			if !cb(d) {
				break
			}
		}
	}
}

func (ws *workspace) ORecord(name QName) IORecord {
	if t := ws.typeByKind(name, TypeKind_ORecord); t != nil {
		return t.(IORecord)
	}
	return nil
}

func (ws *workspace) ORecords(cb func(IORecord) bool) {
	for t := range ws.Types {
		if r, ok := t.(IORecord); ok {
			if !cb(r) {
				break
			}
		}
	}
}

func (ws *workspace) Record(name QName) IRecord {
	if t := ws.TypeByName(name); t != nil {
		if r, ok := t.(IRecord); ok {
			return r
		}
	}
	return nil
}

func (ws *workspace) Records(cb func(IRecord) bool) {
	for t := range ws.Types {
		if r, ok := t.(IRecord); ok {
			if !cb(r) {
				break
			}
		}
	}
}

func (ws *workspace) Singleton(name QName) ISingleton {
	if t := ws.TypeByName(name); t != nil {
		if s, ok := t.(ISingleton); ok {
			return s
		}
	}
	return nil
}

func (ws *workspace) Singletons(cb func(ISingleton) bool) {
	for t := range ws.Types {
		if s, ok := t.(ISingleton); ok {
			if s.Singleton() {
				if !cb(s) {
					break
				}
			}
		}
	}
}

func (ws *workspace) Structure(name QName) IStructure {
	if t := ws.TypeByName(name); t != nil {
		if s, ok := t.(IStructure); ok {
			return s
		}
	}
	return nil
}

func (ws *workspace) Structures(cb func(IStructure) bool) {
	for t := range ws.Types {
		if r, ok := t.(IStructure); ok {
			if !cb(r) {
				break
			}
		}
	}
}

func (ws *workspace) Type(name QName) IType {
	if t := ws.TypeByName(name); t != nil {
		return t
	}
	return NullType
}

func (ws *workspace) TypeByName(name QName) IType {
	if t, ok := ws.types[name]; ok {
		return t.(IType)
	}
	for _, a := range ws.ancestors {
		if t := a.TypeByName(name); t != nil {
			return t
		}
	}
	return nil
}

func (ws *workspace) Types(cb func(IType) bool) {
	if len(ws.typesOrdered) != len(ws.types) {
		ws.typesOrdered = make([]interface{}, 0, len(ws.types))
		for _, t := range ws.types {
			ws.typesOrdered = append(ws.typesOrdered, t)
		}
		sort.Slice(ws.typesOrdered, func(i, j int) bool {
			return ws.typesOrdered[i].(IType).QName().String() < ws.typesOrdered[j].(IType).QName().String()
		})
	}
	for _, t := range ws.typesOrdered {
		if !cb(t.(IType)) {
			break
		}
	}
}

func (ws *workspace) Validate() error {
	if (ws.desc != nil) && ws.desc.Abstract() && !ws.Abstract() {
		return ErrIncompatible("%v should be abstract because descriptor %v is abstract", ws, ws.desc)
	}
	return nil
}

func (ws *workspace) View(name QName) IView {
	if t := ws.typeByKind(name, TypeKind_ViewRecord); t != nil {
		return t.(IView)
	}
	return nil
}

func (ws *workspace) Views(cb func(IView) bool) {
	for t := range ws.Types {
		if v, ok := t.(IView); ok {
			if !cb(v) {
				break
			}
		}
	}
}

func (ws *workspace) WDoc(name QName) IWDoc {
	if t := ws.typeByKind(name, TypeKind_WDoc); t != nil {
		return t.(IWDoc)
	}
	return nil
}

func (ws *workspace) WDocs(cb func(IWDoc) bool) {
	for t := range ws.Types {
		if d, ok := t.(IWDoc); ok {
			if !cb(d) {
				break
			}
		}
	}
}

func (ws *workspace) WRecord(name QName) IWRecord {
	if t := ws.typeByKind(name, TypeKind_WRecord); t != nil {
		return t.(IWRecord)
	}
	return nil
}

func (ws *workspace) WRecords(cb func(IWRecord) bool) {
	for t := range ws.Types {
		if r, ok := t.(IWRecord); ok {
			if !cb(r) {
				break
			}
		}
	}
}

func (ws *workspace) addCDoc(name QName) ICDocBuilder {
	d := newCDoc(ws.app, ws, name)
	return newCDocBuilder(d)
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

// TODO: should be deprecated. All types should be added by specific methods.
func (ws *workspace) addType(name QName) {
	t := ws.app.TypeByName(name)
	if t == nil {
		panic(ErrTypeNotFound(name))
	}

	ws.types[name] = t
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

func (ws *workspace) appendType(t interface{}) {
	ws.app.appendType(t)

	typ := t.(IType)
	name := typ.QName()

	// do not check the validity or uniqueness of the name; this was checked by `*application.appendType (t)`

	ws.types[name] = t
	ws.typesOrdered = nil
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
	}

	clear(ws.ancestors)
	ws.ancestorsOrdered = nil

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

	if ws.desc = ws.app.CDoc(q); ws.desc == nil {
		panic(ErrNotFound("CDoc «%v»", q))
	}
	if ws.desc.Abstract() {
		ws.withAbstract.setAbstract()
	}

	ws.app.wsDesc[q] = ws
}

// Returns type by name and kind. If type is not found then returns nil.
func (ws *workspace) typeByKind(name QName, kind TypeKind) interface{} {
	if t := ws.Type(name); t.Kind() == kind {
		return t
	}
	return nil
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

func (wb *workspaceBuilder) AddCRecord(name QName) ICRecordBuilder {
	return wb.workspace.addCRecord(name)
}

func (wb *workspaceBuilder) AddGDoc(name QName) IGDocBuilder {
	return wb.workspace.addGDoc(name)
}

func (wb *workspaceBuilder) AddGRecord(name QName) IGRecordBuilder {
	return wb.workspace.addGRecord(name)
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

// TODO: should be deprecated. All types should be added by specific methods.
func (wb *workspaceBuilder) AddType(name QName) IWorkspaceBuilder {
	wb.workspace.addType(name)
	return wb
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

func (wb *workspaceBuilder) SetAncestors(name QName, names ...QName) IWorkspaceBuilder {
	wb.workspace.setAncestors(name, names...)
	return wb
}

func (wb *workspaceBuilder) SetDescriptor(q QName) IWorkspaceBuilder {
	wb.workspace.setDescriptor(q)
	return wb
}

func (wb *workspaceBuilder) Workspace() IWorkspace { return wb.workspace }
