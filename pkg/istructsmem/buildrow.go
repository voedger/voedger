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
	switch r := w.(type) {
	case *rowType:
		reader, err = r, r.build()
	case *recordType:
		reader, err = r, r.build()
	case *objectType:
		reader, err = r, r.build()
	case *keyType:
		reader, err = r, r.build()
	case *valueType:
		reader, err = r, r.build()
	default:
		err = errors.ErrUnsupported
	}
	return reader, err
}

var _ = istructs.CollectRowBuilder(buildRow)
