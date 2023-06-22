/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

type cachedAppStorage struct {
	cache                *fastcache.Cache
	keyPool              keyPool
	storage              istorage.IAppStorage
	vvm                  string
	appQName             istructs.AppQName
	mGetSeconds          *float64
	mGetTotal            *float64
	mGetCachedTotal      *float64
	mGetBatchSeconds     *float64
	mGetBatchTotal       *float64
	mGetBatchCachedTotal *float64
	mPutTotal            *float64
	mPutSeconds          *float64
	mPutBatchTotal       *float64
	mPutBatchSeconds     *float64
	mPutBatchItemsTotal  *float64
	mReadTotal           *float64
	mReadSeconds         *float64
}

type implCachingAppStorageProvider struct {
	storageProvider istorage.IAppStorageProvider
	conf            StorageCacheConf
	metrics         imetrics.IMetrics
	vvmName         string
}

func (asp *implCachingAppStorageProvider) AppStorage(appQName istructs.AppQName) (istorage.IAppStorage, error) {
	nonCachingAppStorage, err := asp.storageProvider.AppStorage(appQName)
	if err != nil {
		return nil, err
	}
	return newCachingAppStorage(asp.conf, nonCachingAppStorage, asp.metrics, asp.vvmName, appQName), nil
}

func newCachingAppStorage(conf StorageCacheConf, nonCachingAppStorage istorage.IAppStorage, metrics imetrics.IMetrics, vvm string, appQName istructs.AppQName) istorage.IAppStorage {
	return &cachedAppStorage{
		cache:                fastcache.New(conf.MaxBytes),
		keyPool:              newKeyPool(conf.PreAllocatedBuffersCount, conf.PreAllocatedBufferSize),
		storage:              nonCachingAppStorage,
		vvm:                  vvm,
		appQName:             appQName,
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
	}
}

func (s *cachedAppStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	start := time.Now()
	defer func() {
		imetrics.AddFloat64(s.mPutSeconds, time.Since(start).Seconds())
	}()
	imetrics.AddFloat64(s.mPutTotal, 1.0)
	err = s.storage.Put(pKey, cCols, value)
	if err == nil {
		bb, err := s.keyPool.get(pKey, cCols)
		if err != nil {
			return err
		}
		s.cache.Set(bb.Bytes(), value)
		s.keyPool.put(bb)
	}
	return err
}

func (s *cachedAppStorage) PutBatch(items []istorage.BatchItem) (err error) {
	start := time.Now()
	defer func() {
		imetrics.AddFloat64(s.mPutBatchSeconds, time.Since(start).Seconds())
	}()
	imetrics.AddFloat64(s.mPutBatchTotal, 1.0)
	imetrics.AddFloat64(s.mPutBatchItemsTotal, float64(len(items)))

	err = s.storage.PutBatch(items)
	if err == nil {
		for _, i := range items {
			bb, err := s.keyPool.get(i.PKey, i.CCols)
			if err != nil {
				return err
			}
			s.cache.Set(bb.Bytes(), i.Value)
			s.keyPool.put(bb)
		}
	}
	return err
}

func (s *cachedAppStorage) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	start := time.Now()
	defer func() {
		imetrics.AddFloat64(s.mGetSeconds, time.Since(start).Seconds())
	}()
	imetrics.AddFloat64(s.mGetTotal, 1.0)

	bb, err := s.keyPool.get(pKey, cCols)
	if err != nil {
		return
	}
	defer s.keyPool.put(bb)

	*data = (*data)[0:0]
	*data = s.cache.Get(*data, bb.Bytes())
	if len(*data) != 0 {
		imetrics.AddFloat64(s.mGetCachedTotal, 1.0)
		return true, err
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		bb, err := s.keyPool.get(pKey, cCols)
		if err != nil {
			return false, err
		}
		defer s.keyPool.put(bb)
		s.cache.Set(bb.Bytes(), *data)
	}
	return
}

func (s *cachedAppStorage) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	start := time.Now()
	defer func() {
		imetrics.AddFloat64(s.mGetBatchSeconds, time.Since(start).Seconds())
	}()
	imetrics.AddFloat64(s.mGetBatchTotal, 1.0)
	if !s.getBatchFromCache(pKey, items) {
		return s.getBatchFromStorage(pKey, items)
	}
	return
}

func (s *cachedAppStorage) getBatchFromCache(pKey []byte, items []istorage.GetBatchItem) (ok bool) {
	for i := range items {
		bb, err := s.keyPool.get(pKey, items[i].CCols)
		if err != nil {
			return false
		}
		*items[i].Data = s.cache.Get((*items[i].Data)[0:0], bb.Bytes())
		if len(*items[i].Data) == 0 {
			return false
		}
		items[i].Ok = true
		s.keyPool.put(bb)
	}
	imetrics.AddFloat64(s.mGetBatchCachedTotal, 1.0)
	return true
}

func (s *cachedAppStorage) getBatchFromStorage(pKey []byte, items []istorage.GetBatchItem) (err error) {
	err = s.storage.GetBatch(pKey, items)
	if err != nil {
		return err
	}
	for _, item := range items {
		bb, e := s.keyPool.get(pKey, item.CCols)
		if e != nil {
			return e
		}
		if item.Ok {
			s.cache.Set(bb.Bytes(), *item.Data)
		} else {
			s.cache.Del(bb.Bytes())
		}
		s.keyPool.put(bb)
	}
	return err
}

func (s *cachedAppStorage) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	start := time.Now()
	defer func() {
		imetrics.AddFloat64(s.mReadSeconds, time.Since(start).Seconds())
	}()
	imetrics.AddFloat64(s.mReadTotal, 1.0)

	return s.storage.Read(ctx, pKey, startCCols, finishCCols, cb)
}
