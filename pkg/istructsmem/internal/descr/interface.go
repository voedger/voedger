/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Application struct {
	Name     appdef.AppQName
	Packages map[string]*Package `json:",omitempty"`
}

type Package struct {
	Name       string                      `json:"-"`
	Path       string                      `json:",omitempty"`
	Workspaces map[appdef.QName]*Workspace `json:",omitempty"`
}
