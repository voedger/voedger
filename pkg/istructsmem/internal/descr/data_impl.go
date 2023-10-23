/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newData() *Data {
	return &Data{
		Constraints: make(map[string]any, 0),
	}
}

func (d *Data) read(data appdef.IData) {
	d.Comment = data.Comment()
	if q := data.QName(); q != appdef.NullQName {
		d.Name = &q
	}
	if data.Ancestor() != nil {
		q := data.Ancestor().QName()
		d.Ancestor = &q
	} else {
		k := data.DataKind()
		d.DataKind = &k
	}
	data.Constraints(func(c appdef.IConstraint) {
		d.Constraints[c.Kind().TrimString()] = c.Value()
	})
}
