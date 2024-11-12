/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Data struct {
	Type
	Ancestor    appdef.QName   `json:",omitempty"`
	Constraints map[string]any `json:",omitempty"`
}
