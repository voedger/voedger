/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

type Resource struct {
	Kind    istructs.ResourceKindType
	Name    istructs.QName
	Command *Command `json:",omitempty"`
	Query   *Query   `json:",omitempty"`
}

type Command struct {
	ParamsSchema         *istructs.QName `json:",omitempty"`
	UnloggedParamsSchema *istructs.QName `json:",omitempty"`
	ResultSchema         *istructs.QName `json:",omitempty"`
}

type Query struct {
	ParamsSchema *istructs.QName `json:",omitempty"`
	ResultSchema *istructs.QName `json:",omitempty"`
}
