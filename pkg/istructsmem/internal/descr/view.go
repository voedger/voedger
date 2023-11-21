/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

type View struct {
	Type
	Key   Key
	Value []*Field `json:",omitempty"`
}

type Key struct {
	Partition []*Field
	ClustCols []*Field
}
