/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newData() *Data {
	return &Data{
		Name:     appdef.NullQName,
		DataKind: appdef.DataKind_null,
		Ancestor: appdef.NullQName,
	}
}

func (d *Data) read(data appdef.IData) {
	d.Comment = data.Comment()
	d.Name = data.QName()
	d.DataKind = data.DataKind()
	if data.Ancestor() != nil {
		d.Ancestor = data.Ancestor().QName()
	}
}
