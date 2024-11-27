/*
 *  Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import "errors"

var ErrBLOBNotFound = errors.New("BLOB not found")
var ErrBLOBCorrupted = errors.New("BLOB corrupted")
var ErrBLOBSizeQuotaExceeded = errors.New("BLOB size quote exceeded")
