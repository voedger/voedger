/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newView() *View {
	return &View{
		Key:   Key{},
		Value: make([]*Field, 0),
	}
}

func (v *View) read(view appdef.IView) {
	v.Type.read(view)

	v.Key.read(view.Key())

	for _, fld := range view.Value().Fields() {
		f := newField()
		f.read(fld)
		v.Value = append(v.Value, f)
	}
}

func (k *Key) read(key appdef.IViewKey) {
	for _, fld := range key.PartKey().Fields() {
		f := newField()
		f.read(fld)
		k.Partition = append(k.Partition, f)
	}
	for _, fld := range key.ClustCols().Fields() {
		f := newField()
		f.read(fld)
		k.ClustCols = append(k.ClustCols, f)
	}
}
