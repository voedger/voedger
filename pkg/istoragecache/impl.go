/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

type cachedAppStorage struct {
	cache    *fastcache.Cache
	storage  istorage.IAppStorage
	vvm      string
	appQName appdef.AppQName

	/* metrics */
	mGetSeconds          *imetrics.MetricValue
	mGetTotal            *imetrics.MetricValue
	mGetCachedTotal      *imetrics.MetricValue
	mGetBatchSeconds     *imetrics.MetricValue
	mGetBatchTotal       *imetrics.MetricValue
	mGetBatchCachedTotal *imetrics.MetricValue
	mPutTotal            *imetrics.MetricValue
	mPutSeconds          *imetrics.MetricValue
	mPutBatchTotal       *imetrics.MetricValue
	mPutBatchSeconds     *imetrics.MetricValue
	mPutBatchItemsTotal  *imetrics.MetricValue
	mReadTotal           *imetrics.MetricValue
	mReadSeconds         *imetrics.MetricValue
}

type implCachingAppStorageProvider struct {
	storageProvider istorage.IAppStorageProvider
	maxBytes        int
	metrics         imetrics.IMetrics
	vvmName         string
}

func (asp *implCachingAppStorageProvider) AppStorage(appQName appdef.AppQName) (istorage.IAppStorage, error) {
	nonCachingAppStorage, err := asp.storageProvider.AppStorage(appQName)
	if err != nil {
		return nil, err
	}
	return newCachingAppStorage(asp.maxBytes, nonCachingAppStorage, asp.metrics, asp.vvmName, appQName), nil
}

func newCachingAppStorage(maxBytes int, nonCachingAppStorage istorage.IAppStorage, metrics imetrics.IMetrics, vvm string, appQName appdef.AppQName) istorage.IAppStorage {
	return &cachedAppStorage{
		cache:                fastcache.New(maxBytes),
		storage:              nonCachingAppStorage,
		mGetTotal:            metrics.AppMetricAddr(getTotal, vvm, appQName),
		mGetCachedTotal:      metrics.AppMetricAddr(getCachedTotal, vvm, appQName),
		mGetSeconds:          metrics.AppMetricAddr(getSeconds, vvm, appQName),
		mGetBatchSeconds:     metrics.AppMetricAddr(getBatchSeconds, vvm, appQName),
		mGetBatchTotal:       metrics.AppMetricAddr(getBatchTotal, vvm, appQName),
		mGetBatchCachedTotal: metrics.AppMetricAddr(getBatchCachedTotal, vvm, appQName),
		mPutTotal:            metrics.AppMetricAddr(putTotal, vvm, appQName),
		mPutSeconds:          metrics.AppMetricAddr(putSeconds, vvm, appQName),
		mPutBatchTotal:       metrics.AppMetricAddr(putBatchTotal, vvm, appQName),
		mPutBatchSeconds:     metrics.AppMetricAddr(putBatchSeconds, vvm, appQName),
		mPutBatchItemsTotal:  metrics.AppMetricAddr(putBatchItemsTotal, vvm, appQName),
		mReadTotal:           metrics.AppMetricAddr(readTotal, vvm, appQName),
		mReadSeconds:         metrics.AppMetricAddr(readSeconds, vvm, appQName),
		vvm:                  vvm,
		appQName:             appQName,
	}
}

func (s *cachedAppStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	start := time.Now()
	defer func() {
		s.mPutSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mPutTotal.Increase(1.0)
	err = s.storage.Put(pKey, cCols, value)
	if err == nil {
		s.cache.Set(makeKey(pKey, cCols), value)
	}
	return err
}

func (s *cachedAppStorage) PutBatch(items []istorage.BatchItem) (err error) {
	start := time.Now()
	defer func() {
		s.mPutBatchSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mPutBatchTotal.Increase(1.0)
	s.mPutBatchItemsTotal.Increase(float64(len(items)))

	err = s.storage.PutBatch(items)
	if err == nil {
		for _, i := range items {
			s.cache.Set(makeKey(i.PKey, i.CCols), i.Value)
		}
	}
	return err
}

func (s *cachedAppStorage) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	start := time.Now()
	defer func() {
		s.mGetSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mGetTotal.Increase(1.0)

	key := makeKey(pKey, cCols)

	*data = (*data)[0:0]
	*data, ok = s.cache.HasGet(*data, key)
	if ok {
		s.mGetCachedTotal.Increase(1.0)
		return len(*data) != 0, nil
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		s.cache.Set(key, *data)
	} else {
		s.cache.Set(key, nil)
	}
	return ok, nil
}

func (s *cachedAppStorage) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	start := time.Now()
	defer func() {
		s.mGetBatchSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mGetBatchTotal.Increase(1.0)
	if !s.getBatchFromCache(pKey, items) {
		return s.getBatchFromStorage(pKey, items)
	}
	return
}

func (s *cachedAppStorage) getBatchFromCache(pKey []byte, items []istorage.GetBatchItem) (ok bool) {
	for i := range items {
		*items[i].Data, ok = s.cache.HasGet((*items[i].Data)[0:0], makeKey(pKey, items[i].CCols))
		if !ok {
			return false
		}
		items[i].Ok = len(*items[i].Data) != 0
	}
	s.mGetBatchCachedTotal.Increase(1.0)
	return true
}

func (s *cachedAppStorage) getBatchFromStorage(pKey []byte, items []istorage.GetBatchItem) (err error) {
	err = s.storage.GetBatch(pKey, items)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Ok {
			s.cache.Set(makeKey(pKey, item.CCols), *item.Data)
		} else {
			s.cache.Set(makeKey(pKey, item.CCols), nil)
		}
	}
	return err
}

func (s *cachedAppStorage) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	start := time.Now()
	defer func() {
		s.mReadSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mReadTotal.Increase(1.0)

	return s.storage.Read(ctx, pKey, startCCols, finishCCols, cb)
}

func (s *cachedAppStorage) SetTestDelayGet(delay time.Duration) {
	s.storage.(istorage.IStorageDelaySetter).SetTestDelayGet(delay)
}

func (s *cachedAppStorage) SetTestDelayPut(delay time.Duration) {
	s.storage.(istorage.IStorageDelaySetter).SetTestDelayPut(delay)
}

func makeKey(pKey []byte, cCols []byte) (res []byte) {
	res = make([]byte, 0, stackKeySize)
	// res = make([]byte, 0, len(pKey)+len(cCols)) // escapes to heap
	res = append(res, pKey...)
	res = append(res, cCols...)
	return res
}
