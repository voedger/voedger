/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package iratesce

import (
	irates "github.com/untillpro/voedger/pkg/irates"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

var TestBucketsFactory = func() irates.IBuckets {
	return Provide(coreutils.TestTimeFunc)
}
