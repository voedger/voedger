/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Workspace struct {
	Type
	Descriptor *appdef.QName               `json:",omitempty"`
	Tags       map[appdef.QName]*Tag       `json:",omitempty"`
	DataTypes  map[appdef.QName]*Data      `json:",omitempty"`
	Structures map[appdef.QName]*Structure `json:",omitempty"`
	Views      map[appdef.QName]*View      `json:",omitempty"`
	Extensions *Extensions                 `json:",omitempty"`
	Roles      map[appdef.QName]*Role      `json:",omitempty"`
	ACL        *ACL                        `json:",omitempty"`
	Rates      map[appdef.QName]*Rate      `json:",omitempty"`
	Limits     map[appdef.QName]*Limit     `json:",omitempty"`
}
