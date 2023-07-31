/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//   - IWDoc, IWDocBuilder
type workspace struct {
	def
	withAbstract
	defs map[QName]interface{}
}

func newWorkspace(app *appDef, name QName) *workspace {
	ws := &workspace{
		def:  makeDef(app, name, DefKind_Workspace),
		defs: make(map[QName]interface{}),
	}
	app.appendDef(ws)
	return ws
}

func (ws *workspace) AddDef(name QName) IWorkspaceBuilder {
	d, ok := ws.app.defs[name]
	if !ok {
		panic(fmt.Errorf("unable to add unknown definition «%v» to workspace «%v»: %w", name, ws.QName(), ErrNameNotFound))
	}

	ws.defs[name] = d
	return ws
}

func (ws *workspace) Def(name QName) IDef {
	if d, ok := ws.defs[name]; ok {
		return d.(IDef)
	}
	return nil
}

func (ws *workspace) Defs(cb func(IDef)) {
	for _, d := range ws.defs {
		cb(d.(IDef))
	}
}
