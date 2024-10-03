/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Workspace struct {
	Type
	Descriptor *appdef.QName `json:",omitempty"`
	Types      appdef.QNames `json:",omitempty"`
}
