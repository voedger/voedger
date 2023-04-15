/*
 *  Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import "errors"

//	"context"

var ErrBLOBNotFound = errors.New("BLOB not found")
var ErrBLOBSizeQuotaExceeded = errors.New("BLOB size quote exceeded")
