/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package irates

import "github.com/voedger/voedger/pkg/appdef"

var (
	// NullBucketsFactory is factory what always returns NullBucket
	NullBucketsFactory BucketsFactoryType = func() IBuckets { return NullBucket{} }
)

// # Supports:
//   - IBuckets
//
// NullBucket is a bucket that always full, e.g. TakeTokens always returns true
type NullBucket struct{}

func (NullBucket) TakeTokens([]BucketKey, int) (bool, appdef.QName) { return true, appdef.NullQName }
func (NullBucket) SetDefaultBucketState(appdef.QName, BucketState)  {}
func (NullBucket) GetDefaultBucketsState(appdef.QName) (BucketState, error) {
	return BucketState{}, nil
}
func (NullBucket) ResetRateBuckets(appdef.QName, BucketState)    {}
func (NullBucket) SetBucketState(BucketKey, BucketState) error   { return nil }
func (NullBucket) GetBucketState(BucketKey) (BucketState, error) { return BucketState{}, nil }
