/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// Add new singletone CDoc by specified workspace builder and sets it as workspace descriptor.
// Name is automatically generated, see `DefaultWSDescriptorEntityFmt`
func SetEmptyWSDesc(wsb IWorkspaceBuilder) {
	ws := wsb.Workspace()
	if exists := ws.Descriptor(); exists != NullQName {
		panic(ErrAlreadyExists("%v descriptor %v", ws, exists))
	}
	q := ws.QName()
	n := NewQName(q.Pkg(), fmt.Sprintf(DefaultWSDescriptorEntityFmt, q.Entity()))
	wsb.AddCDoc(n).SetSingleton()
	wsb.SetDescriptor(n)
}
