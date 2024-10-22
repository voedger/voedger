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
	types        map[QName]interface{}
	typesOrdered []interface{}
	desc         ICDoc
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		typ:   makeType(app, name, TypeKind_Workspace),
		types: make(map[QName]interface{}),
	}
	app.appendType(ws)
	return ws
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

// TODO: should be deprecated. All types should be added by specific methods.
func (ws *workspace) addType(name QName) {
	t := ws.app.TypeByName(name)
	if t == nil {
		panic(ErrTypeNotFound(name))
	}

	ws.types[name] = t
}

func (ws *workspace) addData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	d := newData(ws.app, name, kind, ancestor)
	d.addConstraints(constraints...)
	ws.app.appendType(d)
	ws.types[name] = d
	return newDataBuilder(d)
}

func (ws *workspace) addGDoc(name QName) IGDocBuilder {
	gDoc := newGDoc(ws.app, name)
	ws.types[name] = gDoc
	return newGDocBuilder(gDoc)
}

func (ws *workspace) addGRecord(name QName) IGRecordBuilder {
	gRec := newGRecord(ws.app, name)
	ws.types[name] = gRec
	return newGRecordBuilder(gRec)
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
	if t, ok := ws.types[name]; ok {
		if t.(IType).Kind() == kind {
			return t
		}
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

func (wb *workspaceBuilder) AddGDoc(name QName) IGDocBuilder {
	return wb.workspace.addGDoc(name)
}

func (wb *workspaceBuilder) AddGRecord(name QName) IGRecordBuilder {
	return wb.workspace.addGRecord(name)
}

// TODO: should be deprecated. All types should be added by specific methods.
func (wb *workspaceBuilder) AddType(name QName) IWorkspaceBuilder {
	wb.workspace.addType(name)
	return wb
}

func (wb *workspaceBuilder) SetDescriptor(q QName) IWorkspaceBuilder {
	wb.workspace.setDescriptor(q)
	return wb
}

func (wb *workspaceBuilder) Workspace() IWorkspace { return wb.workspace }
