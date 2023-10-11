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
	Command *CommandResource `json:",omitempty"`
	Query   *QueryResource   `json:",omitempty"`
}

type CommandResource struct {
	Params   *appdef.QName `json:",omitempty"`
	Unlogged *appdef.QName `json:",omitempty"`
	Result   *appdef.QName `json:",omitempty"`
}

type QueryResource struct {
	Params *appdef.QName `json:",omitempty"`
	Result *appdef.QName `json:",omitempty"`
}
