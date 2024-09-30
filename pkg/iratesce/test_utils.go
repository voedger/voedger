/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package iratesce

import (
	"github.com/voedger/voedger/pkg/coreutils"
	irates "github.com/voedger/voedger/pkg/irates"
)

var TestBucketsFactory = func() irates.IBuckets {
	return Provide(coreutils.MockTime)
}
