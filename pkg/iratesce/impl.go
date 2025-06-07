/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package iratesce

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
)

// is the bucket's state undefined (zero)
func BucketStateIsZero(state *irates.BucketState) bool {
	return state.TakenTokens == 0 && state.Period == 0 && state.MaxTokensPerPeriod == 0
}

// creating a new bucket with the passed parameters
func newBucket(state irates.BucketState, now time.Time) (p *bucketType) {
	b := bucketType{
		state: state,
	}
	b.reset(now)
	return &b
}

// apply state parameters to the backet
func (bucket *bucketType) resetToState(state irates.BucketState, now time.Time) {
	bucket.state = state
	bucket.reset(now)
}

// recalculates the number of bucket.state tokens.TakenTokens for the time time
func (bucket *bucketType) recalcBuketState(now time.Time) {
	_, _, tokens := bucket.limiter.advance(now)
	value := float64(bucket.limiter.burst) - tokens
	if value < 0 {
		value = 0
	}
	bucket.state.TakenTokens = irates.NumTokensType(value)
}

// reset the bucket to its original state corresponding to the parameters with which it was created (fill it with tokens)
func (bucket *bucketType) reset(now time.Time) {
	var interval Limit
	if bucket.state.MaxTokensPerPeriod > 0 {
		interval = every(time.Duration(int64(bucket.state.Period) / int64(bucket.state.MaxTokensPerPeriod)))
	}
	bucket.limiter = *newLimiter(interval, int(bucket.state.MaxTokensPerPeriod))
	bucket.limiter.allowN(now, int(bucket.state.TakenTokens))
}

// Try to take n tokens from the given buckets
// The operation must be atomic - either all buckets are modified or none
func (b *bucketsType) TakeTokens(buckets []irates.BucketKey, n int) (ok bool, excLimit appdef.QName) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var keyIdx int
	ok = true
	excLimit = appdef.NullQName
	t := b.time.Now()
	// let's check the presence of a token using the requested keys
	for keyIdx = 0; keyIdx < len(buckets); keyIdx++ {
		bucket := b.bucketByKey(&buckets[keyIdx])
		// if for some reason the bucket for the next key is not found, then its absence should not affect the overall result of the check
		// the key may contain the name of the action for which the restriction was not set. In this case, the restriction of this action is not performed
		if bucket == nil {
			continue
		}

		// if the next token is not received, then we leave the request cycle
		if !bucket.limiter.allowN(t, n) {
			ok = false
			excLimit = buckets[keyIdx].RateLimitName
			break
		}

	}

	// if we have not received tokens for all keys, then we will return the tokens taken back to the buckets
	if !ok {
		for i := 0; i < keyIdx; i++ {
			if bucket := b.bucketByKey(&buckets[i]); bucket != nil {
				bucket.limiter.allowN(t, -n)
			}
		}
	}
	return ok, excLimit
}

// returns bucket from the map
// if there is no bucket for the requested key yet, it will be pre-created with the "default" parameters
func (b *bucketsType) bucketByKey(key *irates.BucketKey) (bucket *bucketType) {
	if bucket, ok := b.buckets[*key]; ok {
		return bucket
	}

	// if there is no bucket for the key yet, then we will create it
	bs, ok := b.defaultStates[key.RateLimitName]
	if !ok {
		return nil
	}
	bucket = newBucket(bs, b.time.Now())
	b.buckets[*key] = bucket
	return bucket
}

// at the same time, the working Bucket's parameters of restrictions do not change
// to change the parameters of working buckets, use the ReserRateBuckets function
// setting the "default" constraint parameters for an action named RateLimitName
func (b *bucketsType) SetDefaultBucketState(rateLimitName appdef.QName, bucketState irates.BucketState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.defaultStates[rateLimitName] = bucketState
}

// returns irates.ErrorRateLimitNotFound
func (b *bucketsType) GetDefaultBucketsState(rateLimitName appdef.QName) (state irates.BucketState, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if state, ok := b.defaultStates[rateLimitName]; ok {
		return state, nil
	}
	return state, irates.ErrorRateLimitNotFound
}

// change the restriction parameters with the name RateLimitName for running buckets on bucketState
// the corresponding buckets will be "reset" to the maximum allowed number of available tokens
func (b *bucketsType) ResetRateBuckets(rateLimitName appdef.QName, bucketState irates.BucketState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.defaultStates[rateLimitName]

	// if the "default" parameters for this restriction were not set earlier, then
	// there are definitely no buckets for this restriction. Just leave
	if !ok {
		return
	}

	for bucketKey, bucket := range b.buckets {
		if bucketKey.RateLimitName == rateLimitName {
			bucket.resetToState(bucketState, b.time.Now())
		}
	}
}

// getting the restriction parameters for the bucket corresponding to the transmitted key
func (b *bucketsType) GetBucketState(bucketKey irates.BucketKey) (state irates.BucketState, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	buc := b.bucketByKey(&bucketKey)

	if buc != nil {
		buc.recalcBuketState(b.time.Now())
		return buc.state, nil
	}
	return state, irates.ErrorRateLimitNotFound
}

func (b *bucketsType) SetBucketState(bucketKey irates.BucketKey, state irates.BucketState) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	buc := b.bucketByKey(&bucketKey)

	if buc == nil {
		return irates.ErrorRateLimitNotFound
	}

	buc.resetToState(state, b.time.Now())
	return nil
}
