/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type Resource struct {
	Kind    istructs.ResourceKindType
	Name    schemas.QName
	Command *Command `json:",omitempty"`
	Query   *Query   `json:",omitempty"`
}

type Command struct {
	ParamsSchema         *schemas.QName `json:",omitempty"`
	UnloggedParamsSchema *schemas.QName `json:",omitempty"`
	ResultSchema         *schemas.QName `json:",omitempty"`
}

type Query struct {
	ParamsSchema *schemas.QName `json:",omitempty"`
	ResultSchema *schemas.QName `json:",omitempty"`
}
