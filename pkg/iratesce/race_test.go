/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package iratesce

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestRace_Buckets(t *testing.T) {
	buckets := Provide(timeu.NewITime())

	totalRegLimitName := appdef.NewQName("test", "TotalRegLimitName")
	addrRegLimitName := appdef.NewQName("test", "TotalRegLimitName")

	totalRegKey := irates.BucketKey{
		RateLimitName: totalRegLimitName,
		App:           istructs.AppQName_test1_app1,
		QName:         appdef.NewQName("test", "test"),
		RemoteAddr:    "",
		Workspace:     1,
	}

	totalRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 100,
		TakenTokens:        0,
	}

	addrRegKey := irates.BucketKey{
		RateLimitName: addrRegLimitName,
		App:           istructs.AppQName_test1_app1,
		QName:         appdef.NewQName("test", "test"),
		RemoteAddr:    "addr",
		Workspace:     1,
	}

	addrRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 120,
		TakenTokens:        0,
	}

	buckets.SetDefaultBucketState(totalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(addrRegLimitName, addrRegistrationQuota)

	keys := []irates.BucketKey{totalRegKey, addrRegKey}
	var finish sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	start := sync.WaitGroup{}

	var getTokensForRegistration = func() {
		defer finish.Done()
		start.Done()
		for ctx.Err() == nil {
			_, _ = buckets.TakeTokens(keys, 1)
		}
	}

	var getBucketState = func() {
		defer finish.Done()
		start.Done()
		for ctx.Err() == nil {
			_, err := buckets.GetBucketState(totalRegKey)
			require.NoError(t, err)
		}
	}

	var setBucketState = func() {
		defer finish.Done()
		start.Done()
		for ctx.Err() == nil {
			buckets.SetDefaultBucketState(totalRegLimitName, totalRegistrationQuota)
		}
	}

	for i := 0; i < 10; i++ {
		start.Add(3)
		finish.Add(3)
		go getTokensForRegistration()
		go getBucketState()
		go setBucketState()
	}

	start.Wait()
	time.Sleep(time.Second)
	cancel()
	finish.Wait()
}
