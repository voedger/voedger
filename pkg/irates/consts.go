/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package irates

var (
	// NullBucketsFactory is factory what always returns NullBucket
	NullBucketsFactory BucketsFactoryType = func() IBuckets { return NullBucket{} }
)

// # Supports:
//   - IBuckets
//
// NullBucket is a bucket that always full, e.g. TakeTokens always returns true
type NullBucket struct{}

func (NullBucket) TakeTokens([]BucketKey, int) bool                   { return true }
func (NullBucket) SetDefaultBucketState(string, BucketState)          {}
func (NullBucket) GetDefaultBucketsState(string) (BucketState, error) { return BucketState{}, nil }
func (NullBucket) ResetRateBuckets(string, BucketState)               {}
func (NullBucket) SetBucketState(BucketKey, BucketState) error        { return nil }
func (NullBucket) GetBucketState(BucketKey) (BucketState, error)      { return BucketState{}, nil }
