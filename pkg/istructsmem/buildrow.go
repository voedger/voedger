/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"

	"github.com/voedger/voedger/pkg/istructs"
)

func buildRow(w istructs.IRowWriter) (reader istructs.IRowReader, err error) {

	if r, ok := w.(*rowType); ok {
		reader = r
		err = r.build()
	} else if r, ok := w.(*recordType); ok {
		reader = r
		err = r.build()
	} else if r, ok := w.(*objectType); ok {
		reader = r
		err = r.build()
	} else if r, ok := w.(*keyType); ok {
		reader = r
		err = r.build()
	} else if r, ok := w.(*valueType); ok {
		reader = r
		err = r.build()
	} else {
		err = errors.ErrUnsupported
	}

	if err != nil {
		reader = nil
	}
	return reader, err
}

var _ = istructs.CollectRowBuilder(buildRow)
