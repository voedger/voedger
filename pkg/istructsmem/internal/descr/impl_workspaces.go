/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newWorkspace() *Workspace {
	return &Workspace{}
}

func (w *Workspace) read(workspace appdef.IWorkspace) {
	w.Type.read(workspace)
	if name := workspace.Descriptor(); name != appdef.NullQName {
		w.Descriptor = &name
	}
	for t := range workspace.Types {
		w.Types = append(w.Types, t.QName())
	}
}
