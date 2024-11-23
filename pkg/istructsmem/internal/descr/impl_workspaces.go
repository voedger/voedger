/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newWorkspace() *Workspace {
	return &Workspace{
		DataTypes:  make(map[appdef.QName]*Data),
		Structures: make(map[appdef.QName]*Structure),
		Views:      make(map[appdef.QName]*View),
		Roles:      make(map[appdef.QName]*Role),
		ACL:        newACL(),
		Rates:      make(map[appdef.QName]*Rate),
		Limits:     make(map[appdef.QName]*Limit),
	}
}

func (w *Workspace) read(workspace appdef.IWorkspace) {
	w.Type.read(workspace)
	if name := workspace.Descriptor(); name != appdef.NullQName {
		w.Descriptor = &name
	}
	for typ := range workspace.LocalTypes {
		name := typ.QName()

		switch t := typ.(type) {
		case appdef.IData:
			d := newData()
			d.read(t)
			w.DataTypes[name] = d
		case appdef.IStructure:
			s := newStructure()
			s.read(t)
			w.Structures[name] = s
		case appdef.IView:
			v := newView()
			v.read(t)
			w.Views[name] = v
		case appdef.IExtension:
			if w.Extensions == nil {
				w.Extensions = newExtensions()
			}
			w.Extensions.read(t)
		case appdef.IRole:
			r := newRole()
			r.read(t)
			w.Roles[name] = r
		case appdef.IRate:
			r := newRate()
			r.read(t)
			w.Rates[name] = r
		case appdef.ILimit:
			r := newLimit()
			r.read(t)
			w.Limits[name] = r
		}
	}
	w.ACL.read(workspace, true)
}
