/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package workspaces

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
)

// # Supports:
//   - appdef.IWithWorkspaces
type WithWorkspaces struct {
	list    *Workspaces
	changed bool
}

func MakeWithWorkspaces() WithWorkspaces {
	return WithWorkspaces{list: NewWorkspaces()}
}

func (ww *WithWorkspaces) AppendWorkspace(ws appdef.IWorkspace) { ww.list.Add(ws) }

func (ww *WithWorkspaces) Build() (err error) {
	for _, ws := range ww.Workspaces() {
		err = errors.Join(err,
			ws.(*Workspace).build())
	}
	return err
}

// Should be called after successfully built.
func (ww *WithWorkspaces) Builded() {
	for _, ws := range ww.Workspaces() {
		ws.(*Workspace).builded()
	}
	ww.changed = false
}

func (ww *WithWorkspaces) Changed() {
	if !ww.changed {
		ww.changed = true
		for _, ws := range ww.Workspaces() {
			ws.(*Workspace).changed()
		}
	}
}

func (ww WithWorkspaces) Workspace(name appdef.QName) appdef.IWorkspace {
	ws := ww.list.Find(name)
	if ws != appdef.NullType {
		return ws.(appdef.IWorkspace)
	}
	return nil
}

func (ww WithWorkspaces) WorkspaceByDescriptor(desc appdef.QName) appdef.IWorkspace {
	for _, ws := range ww.Workspaces() {
		if ws.Descriptor() == desc {
			return ws
		}
	}
	return nil
}

func (ww WithWorkspaces) Workspaces() []appdef.IWorkspace { return ww.list.AsArray() }

// # Supports:
//   - appdef.IWorkspacesBuilder
type WorkspacesBuilder struct {
	app appdef.IAppDef
	ww  *WithWorkspaces
}

func MakeWorkspacesBuilder(app appdef.IAppDef, ww *WithWorkspaces) WorkspacesBuilder {
	return WorkspacesBuilder{app, ww}
}

func (wb *WorkspacesBuilder) AddWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	ws := NewWorkspace(wb.app, name)
	wb.ww.list.Add(ws)
	return NewWorkspaceBuilder(ws)
}

func (wb *WorkspacesBuilder) AlterWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	ws := wb.ww.Workspace(name)
	if ws == nil {
		panic(appdef.ErrNotFound("workspace «%v»", name))
	}
	return NewWorkspaceBuilder(ws.(*Workspace))
}

func AddWorkspace(app appdef.IAppDef, ww *WithWorkspaces, name appdef.QName) *Workspace {
	ws := NewWorkspace(app, name)
	ww.list.Add(ws)
	return ws
}
