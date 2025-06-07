/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package irates

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Interface should be obtained once
//
// Idea behind: https://en.wikipedia.org/wiki/Token_bucket
// A token is added to the bucket every 1/r seconds.
// The bucket can hold at the most b tokens
// When a packet of n bytes arrives:
// - if at least n tokens are in the bucket, n tokens are removed from the bucket, and the packet is sent to the network.
// - if fewer than n tokens are available, no tokens are removed from the bucket, and the packet is considered to be non-conformant.
type IBuckets interface {

	// Try to take n tokens from the given buckets
	// The operation must be atomic - either all buckets are modified or none
	// If ResetRateBuckets/SetBucketState for the given RateLimitName have not been called true is returned
	//
	// #3027: If returns false, then excLimit is the first RateLimitName that has been exceeded
	TakeTokens(bucketKeys []BucketKey, n int) (ok bool, excLimit appdef.QName)

	SetDefaultBucketState(RateLimitName appdef.QName, bucketState BucketState)

	// returns ErrorRateLimitNotFound
	GetDefaultBucketsState(RateLimitName appdef.QName) (state BucketState, err error)

	// Reset all buckets with given RateLimitName to given state
	ResetRateBuckets(RateLimitName appdef.QName, bucketState BucketState)

	// returns ErrorRateLimitNotFound
	SetBucketState(bucketKey BucketKey, state BucketState) (err error)

	// returns ErrorRateLimitNotFound
	GetBucketState(bucketKey BucketKey) (state BucketState, err error)
}

type BucketKey struct {
	RateLimitName appdef.QName
	RemoteAddr    string
	App           appdef.AppQName //?
	Workspace     istructs.WSID
	QName         appdef.QName
	ID            istructs.RecordID //?
}

type BucketState struct {
	Period             appdef.RatePeriod
	MaxTokensPerPeriod NumTokensType
	TakenTokens        NumTokensType
}

type NumTokensType = appdef.RateCount

type BucketsFactoryType func() IBuckets
