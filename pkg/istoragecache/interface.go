/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package istoragecache

import "bytes"

type keyPool interface {
	get(pKey []byte, cCols []byte) (bb *bytes.Buffer, err error)
	put(bb *bytes.Buffer)
}
