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
	irates "github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestRace_Buckets(t *testing.T) {
	buckets := Provide(time.Now)

	sTotalRegLimitName := "TotalRegLimitName"
	sAddrRegLimitName := "TotalRegLimitName"

	totalRegKey := irates.BucketKey{
		RateLimitName: sTotalRegLimitName,
		App:           istructs.AppQName_test1_app1,
		QName:         schemas.NewQName("test", "test"),
		RemoteAddr:    "",
		Workspace:     1,
	}

	totalRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 100,
		TakenTokens:        0,
	}

	addrRegKey := irates.BucketKey{
		RateLimitName: sAddrRegLimitName,
		App:           istructs.AppQName_test1_app1,
		QName:         schemas.NewQName("test", "test"),
		RemoteAddr:    "addr",
		Workspace:     1,
	}

	addrRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 120,
		TakenTokens:        0,
	}

	buckets.SetDefaultBucketState(sTotalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(sAddrRegLimitName, addrRegistrationQuota)

	keys := []irates.BucketKey{totalRegKey, addrRegKey}
	var finish sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	start := sync.WaitGroup{}

	var getTokensForRegistration = func() {
		defer finish.Done()
		start.Done()
		for ctx.Err() == nil {
			_ = buckets.TakeTokens(keys, 1)
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
			buckets.SetDefaultBucketState(sTotalRegLimitName, totalRegistrationQuota)
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
