/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
  * @author: Dmitry Molchanovsky
*/

package iratesce

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	irates "github.com/voedger/voedger/pkg/irates"
)

// Provide: constructs bucketFactory
func Provide(time timeu.ITime) (buckets irates.IBuckets) {
	return &bucketsType{
		buckets:       map[irates.BucketKey]*bucketType{},
		defaultStates: map[appdef.QName]irates.BucketState{},
		time:          time,
	}
}
