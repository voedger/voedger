/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Application struct {
	Name     appdef.AppQName
	Packages map[string]*Package `json:",omitempty"`
	ACL      *ACL                `json:",omitempty"`
}

type Package struct {
	Name       string                  `json:"-"`
	Path       string                  `json:",omitempty"`
	DataTypes  map[string]*Data        `json:",omitempty"`
	Structures map[string]*Structure   `json:",omitempty"`
	Views      map[string]*View        `json:",omitempty"`
	Extensions *Extensions             `json:",omitempty"`
	Roles      map[string]*Role        `json:",omitempty"`
	Workspaces map[string]*Workspace   `json:",omitempty"`
	Resources  map[string]*Resource    `json:",omitempty"`
	RateLimits map[string][]*RateLimit `json:",omitempty"`
}
