/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"sort"
)

// # Implements:
//   - IWorkspace, IWorkspaceBuilder
type workspace struct {
	typ
	comment
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

func (ws *workspace) AddType(name QName) IWorkspaceBuilder {
	t := ws.app.TypeByName(name)
	if t == nil {
		panic(fmt.Errorf("unable to add unknown type «%v» to workspace «%v»: %w", name, ws.QName(), ErrNameNotFound))
	}

	ws.types[name] = t
	ws.typesOrdered = nil
	return ws
}

func (ws *workspace) Descriptor() QName {
	if ws.desc != nil {
		return ws.desc.QName()
	}
	return NullQName
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

func (ws *workspace) TypeCount() int {
	return len(ws.types)
}

func (ws *workspace) Types(cb func(IType)) {
	if ws.typesOrdered == nil {
		ws.typesOrdered = make([]interface{}, 0, len(ws.types))
		for _, t := range ws.types {
			ws.typesOrdered = append(ws.typesOrdered, t)
		}
		sort.Slice(ws.typesOrdered, func(i, j int) bool {
			return ws.typesOrdered[i].(IType).QName().String() < ws.typesOrdered[j].(IType).QName().String()
		})
	}
	for _, t := range ws.typesOrdered {
		cb(t.(IType))
	}
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
