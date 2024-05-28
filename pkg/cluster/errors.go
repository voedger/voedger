/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import "errors"

var (
	ErrNumPartitionsChanged    = errors.New("num partitions changed")
	ErrNumAppWorkspacesChanged = errors.New("num application workspaces changed")
	errWrongWhereForView       = errors.New("'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported")
	errNullValueNoSupported    = errors.New("null value is not supported")
)
