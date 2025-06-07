/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func (t *Type) read(typ appdef.IType) {
	t.Comment = readComment(typ)
	t.QName = typ.QName()
	t.Kind = typ.Kind()

	t.Tags = appdef.QNames{}
	for _, tag := range typ.Tags() {
		t.Tags.Add(tag.QName())
	}
}
