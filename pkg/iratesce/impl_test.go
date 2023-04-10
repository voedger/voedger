/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package iratesce

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	irates "github.com/untillpro/voedger/pkg/irates"
	"github.com/untillpro/voedger/pkg/istructs"
)

// пример ограничения общего количества регистраций (не более 1000) и регистраций с одного адреса (10) в день
// example of limiting the total number of registrations (no more than 1000) and registrations from one address (10) per day
func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	// описание приложения и рабочей области
	// description of the application and workspace
	app := istructs.AppQName_test1_app1
	qName := istructs.NewQName("testPkg", "test")
	wsid := istructs.WSID(1)

	// имена ограничений
	// constraint names
	sTotalRegLimitName := "TotalRegPerDay"
	sAddrRegLimitName := "AddrRegPerDay"

	// параметры общего ограничения количества регистраций (не более 1000 в сутки)
	// parameters of the general limitation of the number of registrations (no more than 1000 per day)
	totalRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 1000,
		TakenTokens:        0,
	}

	// параметры ограничения количества регистраций с одного адреса (не более 10 в сутки)
	// parameters for limiting the number of registrations from one address (no more than 10 per day)
	addrRegistrationQuota := irates.BucketState{
		Period:             24 * time.Hour,
		MaxTokensPerPeriod: 10,
		TakenTokens:        0,
	}

	// создаем bucket'ы
	// creating buckets
	buckets := Provide(testTimeFunc)

	// передадим в bucket'ы именованые ограничения
	// passing named constraints to bucket's
	buckets.SetDefaultBucketState(sTotalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(sAddrRegLimitName, addrRegistrationQuota)

	state, err := buckets.GetDefaultBucketsState(sTotalRegLimitName)
	require.NoError(err)
	require.Equal(totalRegistrationQuota, state)
	state, err = buckets.GetDefaultBucketsState(sAddrRegLimitName)
	require.NoError(err)
	require.Equal(addrRegistrationQuota, state)

	_, err = buckets.GetDefaultBucketsState("unknown")
	require.ErrorIs(irates.ErrorRateLimitNotFound, err)

	// в процессе работы приложения с некого адреса "remote_address" пришел запрос на регистрацию
	// проверим, доступна ли эта операция
	// during the operation of the application, a registration request came from a certain address "remote_address"
	// let's check if this operation is available

	// ключ для проверки общего количества регистраций
	// key for checking the total number of registrations
	totalRegKey := irates.BucketKey{
		RateLimitName: sTotalRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "",
		Workspace:     wsid,
	}

	// ключ для проверки количества регистраций с адреса "remote_address"
	// key for checking the number of registrations from the address "remote_address"
	addrRegKey := irates.BucketKey{
		RateLimitName: sAddrRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "remote_address",
		Workspace:     wsid,
	}

	// теперь проверим, хватает ли токенов для обоих этих ключей и, если хватает, то разрешим операцию регистрации
	// now let's check if there are enough tokens for both of these keys and, if there are enough, then we will allow the registration operation
	keys := []irates.BucketKey{totalRegKey, addrRegKey}

	// basic usage
	require.True(buckets.TakeTokens(keys, 10))

	// failed to get more tokens because no more tokens for the current moment
	require.False(buckets.TakeTokens(keys, 1))
}

var testTime = time.Now()

var testTimeFunc = func() time.Time {
	return testTime
}

// быстрый тест функционала структуры buckets (реализации интерфейса irates.IBuckets)
// для эмуляции временных диапазонеов используется функция testTimeFunc()
// quick test of the functionality of the buckets structure (implementation of the irates interface.IBuckets)
// the testTimeFunc() function is used to emulate time ranges
func TestBucketsNew(t *testing.T) {
	require := require.New(t)

	buckets := Provide(testTimeFunc)

	// описание приложения и рабочей области
	// description of the application and workspace
	app := istructs.AppQName_test1_app1
	qName := istructs.NewQName("testPkg", "test")
	wsid := istructs.WSID(1)

	// имена ограничений
	// constraint names
	sTotalRegLimitName := "TotalRegPerDay"
	sAddrRegLimitName := "AddrRegPerDay"

	// параметры общего ограничения количества регистраций (не более 100 в час)
	// parameters of the general limitation of the number of registrations (no more than 1000 per hour)
	totalRegistrationQuota := irates.BucketState{
		Period:             1 * time.Hour,
		MaxTokensPerPeriod: 100,
		TakenTokens:        0,
	}

	// параметры ограничения количества регистраций с одного адреса (не более 10 в час)
	// parameters for limiting the number of registrations from one address (no more than 10 per hour)
	addrRegistrationQuota := irates.BucketState{
		Period:             1 * time.Hour,
		MaxTokensPerPeriod: 10,
		TakenTokens:        0,
	}

	// ключ для проверки количества регистраций с адреса "remote_address"
	// key for checking the number of registrations from the address "remote_address"
	addrRegKey := irates.BucketKey{
		RateLimitName: sAddrRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "remote_address",
		Workspace:     wsid,
	}

	// ключ для проверки общего количества регистраций
	// key for checking the total number of registrations
	totalRegKey := irates.BucketKey{
		RateLimitName: sTotalRegLimitName,
		App:           app,
		QName:         qName,
		RemoteAddr:    "",
		Workspace:     wsid,
	}

	buckets.SetDefaultBucketState(sTotalRegLimitName, totalRegistrationQuota)
	buckets.SetDefaultBucketState(sAddrRegLimitName, addrRegistrationQuota)

	keys := []irates.BucketKey{totalRegKey}

	require.True(buckets.TakeTokens(keys, 100))
	bs, err := buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(100))
	require.False(buckets.TakeTokens(keys, 100))
	require.Nil(err)

	testTime = testTime.Add(time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	testTime = testTime.Add(-time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(100))

	testTime = testTime.Add(time.Hour)

	keys = []irates.BucketKey{totalRegKey, addrRegKey}
	require.True(buckets.TakeTokens(keys, 5))
	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))

	require.False(buckets.TakeTokens(keys, 10))
	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(5))

	testTime = testTime.Add(5 * time.Hour)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	require.True(buckets.TakeTokens(keys, 10))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))

	addrRegistrationQuota.MaxTokensPerPeriod = 20
	err = buckets.SetBucketState(addrRegKey, addrRegistrationQuota)
	require.Nil(err)

	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))
	bs, err = buckets.GetBucketState(addrRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	buckets.SetDefaultBucketState(sTotalRegLimitName, totalRegistrationQuota)
	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(10))

	buckets.ResetRateBuckets(sTotalRegLimitName, totalRegistrationQuota)
	bs, err = buckets.GetBucketState(totalRegKey)
	require.Nil(err)
	require.Equal(bs.TakenTokens, irates.NumTokensType(0))

	totalRegKey.RateLimitName = "new limit name"
	bs, err = buckets.GetBucketState(totalRegKey)
	require.NotNil(err)
	require.True(BucketStateIsZero(&bs))
}
