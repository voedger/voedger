/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newTag() *Tag {
	return &Tag{}
}

func (t *Tag) read(tag appdef.ITag) {
	t.Type.read(tag)
	t.Feature = tag.Feature()
}
