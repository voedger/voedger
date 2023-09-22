/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//   - IWDoc, IWDocBuilder
type workspace struct {
	typ
	comment
	withAbstract
	defs map[QName]interface{}
	desc ICDoc
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		typ:  makeType(app, name, TypeKind_Workspace),
		defs: make(map[QName]interface{}),
	}
	app.appendType(ws)
	return ws
}

func (ws *workspace) AddType(name QName) IWorkspaceBuilder {
	d, ok := ws.app.types[name]
	if !ok {
		panic(fmt.Errorf("unable to add unknown type «%v» to workspace «%v»: %w", name, ws.QName(), ErrNameNotFound))
	}

	ws.defs[name] = d
	return ws
}

func (ws *workspace) Type(name QName) IType {
	if d, ok := ws.defs[name]; ok {
		return d.(IType)
	}
	return nil
}

func (ws *workspace) Types(cb func(IType)) {
	for _, d := range ws.defs {
		cb(d.(IType))
	}
}

func (ws *workspace) Descriptor() QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return NullQName
}

func (ws *workspace) SetDescriptor(q QName) IWorkspaceBuilder {
	if ws.desc = ws.app.CDoc(q); ws.desc == nil {
		panic(fmt.Errorf("type «%v» is unknown CDoc name to assign as descriptor for workspace «%v»: %w", q, ws.QName(), ErrNameNotFound))
	}
	if ws.desc.Abstract() {
		ws.SetAbstract()
	}
	return ws
}

func (ws *workspace) Validate() error {
	if (ws.desc != nil) && ws.desc.Abstract() && !ws.Abstract() {
		return fmt.Errorf("workspace %q should be abstract because descriptor %q is abstract: %w", ws.QName(), ws.desc.QName(), ErrWorkspaceShouldBeAbstract)
	}
	return nil
}
