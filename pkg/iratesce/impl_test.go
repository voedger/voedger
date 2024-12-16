/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package iratesce

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

// example of limiting the total number of registrations (no more than 1000) and registrations from one address (10) per day
func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	// description of the application and workspace
	app := istructs.AppQName_test1_app1
	qName := appdef.NewQName("test", "test")
	wsid := istructs.WSID(1)

	// constraint names
	totalRegLimitName := appdef.NewQName("test", "TotalRegPerDay")
	addrRegLimitName := appdef.NewQName("test", "AddrRegPerDay")

	// parameters of the general limitation of the number of registrations (no more than 1000 per day)
	totalRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 1000,
		TakenTokens:        0,
	}

	// parameters for limiting the number of registrations from one address (no more than 10 per day)
	addrRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 10,
		TakenTokens:        0,
	}

	// creating buckets
	buckets := Provide(coreutils.MockTime)

	// passing named constraints to bucket's
	buckets.SetDefaultBucketState(totalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(addrRegLimitName, addrRegistrationQuota)

	state, err := buckets.GetDefaultBucketsState(totalRegLimitName)
	require.NoError(err)
	require.Equal(totalRegistrationQuota, state)
	state, err = buckets.GetDefaultBucketsState(addrRegLimitName)
	require.NoError(err)
	require.Equal(addrRegistrationQuota, state)

	_, err = buckets.GetDefaultBucketsState(appdef.NewQName("test", "unknown"))
	require.ErrorIs(irates.ErrorRateLimitNotFound, err)

	// let's check if this operation is available

	// key for checking the total number of registrations
	totalRegKey := irates.BucketKey{
		RateLimitName: totalRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "",
		Workspace:     wsid,
	}

	// key for checking the number of registrations from the address "remote_address"
	addrRegKey := irates.BucketKey{
		RateLimitName: addrRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "remote_address",
		Workspace:     wsid,
	}

	// now let's check if there are enough tokens for both of these keys and, if there are enough, then we will allow the registration operation
	keys := []irates.BucketKey{totalRegKey, addrRegKey}

	// basic usage
	require.True(buckets.TakeTokens(keys, 10))

	// failed to get more tokens because no more tokens for the current moment
	require.False(buckets.TakeTokens(keys, 1))
}

// quick test of the functionality of the buckets structure (implementation of the irates interface.IBuckets)
func TestBucketsNew(t *testing.T) {
	require := require.New(t)

	buckets := Provide(coreutils.MockTime)

	// description of the application and workspace
	app := istructs.AppQName_test1_app1
	qName := appdef.NewQName("test", "test")
	wsid := istructs.WSID(1)

	// constraint names
	totalRegLimitName := appdef.NewQName("test", "TotalRegPerDay")
	addrRegLimitName := appdef.NewQName("test", "AddrRegPerDay")

	// parameters of the general limitation of the number of registrations (no more than 1000 per hour)
	totalRegistrationQuota := irates.BucketState{
		Period:             1 * time.Hour,
		MaxTokensPerPeriod: 100,
		TakenTokens:        0,
	}

	// parameters for limiting the number of registrations from one address (no more than 10 per hour)
	addrRegistrationQuota := irates.BucketState{
		Period:             1 * time.Hour,
		MaxTokensPerPeriod: 10,
		TakenTokens:        0,
	}

	// key for checking the number of registrations from the address "remote_address"
	addrRegKey := irates.BucketKey{
		RateLimitName: addrRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "remote_address",
		Workspace:     wsid,
	}

	// key for checking the total number of registrations
	totalRegKey := irates.BucketKey{
		RateLimitName: totalRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "",
		Workspace:     wsid,
	}

	buckets.SetDefaultBucketState(totalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(addrRegLimitName, addrRegistrationQuota)

	keys := []irates.BucketKey{totalRegKey}

	require.True(buckets.TakeTokens(keys, 100))
	bs, err := buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(100))
	require.False(buckets.TakeTokens(keys, 100))
	require.NoError(err)

	coreutils.MockTime.Add(time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	coreutils.MockTime.Add(-time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(100))

	coreutils.MockTime.Add(time.Hour)

	keys = []irates.BucketKey{totalRegKey, addrRegKey}
	require.True(buckets.TakeTokens(keys, 5))
	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))

	require.False(buckets.TakeTokens(keys, 10))
	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))

	coreutils.MockTime.Add(5 * time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	require.True(buckets.TakeTokens(keys, 10))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))

	addrRegistrationQuota.MaxTokensPerPeriod = 20
	err = buckets.SetBucketState(addrRegKey, addrRegistrationQuota)
	require.NoError(err)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	buckets.SetDefaultBucketState(totalRegLimitName, totalRegistrationQuota)
	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))

	buckets.ResetRateBuckets(totalRegLimitName, totalRegistrationQuota)
	bs, err = buckets.GetBucketState(totalRegKey)
	require.NoError(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	totalRegKey.RateLimitName = appdef.NewQName("test", "newLimit")
	bs, err = buckets.GetBucketState(totalRegKey)
	require.Error(err)
	require.True(BucketStateIsZero(&bs))
}
