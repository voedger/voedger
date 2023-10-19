/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Data struct {
	Comment  string `json:",omitempty"`
	Name     appdef.QName
	DataKind appdef.DataKind
	Ancestor appdef.QName
}
