/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type Resource struct {
	Kind    istructs.ResourceKindType
	Name    appdef.QName
	Command *Command `json:",omitempty"`
	Query   *Query   `json:",omitempty"`
}

type Command struct {
	Params   *appdef.QName `json:",omitempty"`
	Unlogged *appdef.QName `json:",omitempty"`
	Result   *appdef.QName `json:",omitempty"`
}

type Query struct {
	Params *appdef.QName `json:",omitempty"`
	Result *appdef.QName `json:",omitempty"`
}
