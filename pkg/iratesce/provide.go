/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
  * @author: Dmitry Molchanovsky
*/

package iratesce

import (
	"time"

	irates "github.com/voedger/voedger/pkg/irates"
)

// Provide: constructs bucketFactory
func Provide(timeFunc func() time.Time) (buckets irates.IBuckets) {
	return &bucketsType{
		buckets:       map[irates.BucketKey]*bucketType{},
		defaultStates: map[string]irates.BucketState{},
		timeFunc:      timeFunc,
	}
}
