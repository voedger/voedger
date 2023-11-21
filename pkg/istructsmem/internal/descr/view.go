/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type View struct {
	Comment string       `json:",omitempty"`
	QName   appdef.QName `json:"-"`
	Key     Key
	Value   []*Field `json:",omitempty"`
}

type Key struct {
	Partition []*Field
	ClustCols []*Field
}
