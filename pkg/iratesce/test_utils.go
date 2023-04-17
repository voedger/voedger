/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package iratesce

import (
	irates "github.com/voedger/voedger/pkg/irates"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

var TestBucketsFactory = func() irates.IBuckets {
	return Provide(coreutils.TestTimeFunc)
}
