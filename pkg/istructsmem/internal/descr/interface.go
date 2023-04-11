/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/untillpro/voedger/pkg/istructs"

type Application struct {
	Name     istructs.AppQName
	Packages map[string]*Package
}

type Package struct {
	Name       string
	Schemas    map[string]*Schema      `json:",omitempty"`
	Resources  map[string]*Resource    `json:",omitempty"`
	RateLimits map[string][]*RateLimit `json:",omitempty"`
	Uniques    map[string][]*Unique    `json:",omitempty"`
}
