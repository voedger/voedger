/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
  * @author: Dmitry Molchanovsky
*/

package iratesce

import (
	irates "github.com/voedger/voedger/pkg/irates"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// Provide: constructs bucketFactory
func Provide(timeFunc coreutils.TimeFunc) (buckets irates.IBuckets) {
	return &bucketsType{
		buckets:       map[irates.BucketKey]*bucketType{},
		defaultStates: map[string]irates.BucketState{},
		timeFunc:      timeFunc,
	}
}
