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
	v.Comment = readComment(view)

	v.QName = view.QName()

	v.Key.read(view.Key())

	view.Value().Fields(func(field appdef.IField) {
		f := newField()
		f.read(field)
		v.Value = append(v.Value, f)
	})
}

func (k *Key) read(key appdef.IViewKey) {
	key.PartKey().Fields(func(field appdef.IField) {
		f := newField()
		f.read(field)
		k.Partition = append(k.Partition, f)
	})
	key.ClustCols().Fields(func(field appdef.IField) {
		f := newField()
		f.read(field)
		k.ClustCols = append(k.ClustCols, f)
	})
}
