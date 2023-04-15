/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	istorage "github.com/untillpro/voedger/pkg/istorage"
	istructs "github.com/untillpro/voedger/pkg/istructs"
	imetrics "github.com/untillpro/voedger/pkg/metrics"
)

type cachedAppStorage struct {
	cache    *fastcache.Cache
	storage  istorage.IAppStorage
	metrics  imetrics.IMetrics
	hvm      string
	appQName istructs.AppQName
}

type implCachingAppStorageProvider struct {
	storageProvider istorage.IAppStorageProvider
	maxBytes        int
	metrics         imetrics.IMetrics
	hvmName         string
}

func (asp *implCachingAppStorageProvider) AppStorage(appQName istructs.AppQName) (istorage.IAppStorage, error) {
	nonCachingAppStorage, err := asp.storageProvider.AppStorage(appQName)
	if err != nil {
		return nil, err
	}
	return newCachingAppStorage(asp.maxBytes, nonCachingAppStorage, asp.metrics, asp.hvmName, appQName), nil
}

func newCachingAppStorage(maxBytes int, nonCachingAppStorage istorage.IAppStorage, metrics imetrics.IMetrics, hvm string, appQName istructs.AppQName) istorage.IAppStorage {
	return &cachedAppStorage{
		cache:    fastcache.New(maxBytes),
		storage:  nonCachingAppStorage,
		metrics:  metrics,
		hvm:      hvm,
		appQName: appQName,
	}
}

func (s *cachedAppStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	start := time.Now()
	defer func() {
		s.metrics.IncreaseApp(putSeconds, s.hvm, s.appQName, time.Since(start).Seconds())
	}()
	s.metrics.IncreaseApp(putTotal, s.hvm, s.appQName, 1.0)

	err = s.storage.Put(pKey, cCols, value)
	if err == nil {
		s.cache.Set(key(pKey, cCols), value)
	}
	return err
}

func (s *cachedAppStorage) PutBatch(items []istorage.BatchItem) (err error) {
	start := time.Now()
	defer func() {
		s.metrics.IncreaseApp(putBatchSeconds, s.hvm, s.appQName, time.Since(start).Seconds())
	}()
	s.metrics.IncreaseApp(putBatchTotal, s.hvm, s.appQName, 1.0)
	s.metrics.IncreaseApp(putBatchItemsTotal, s.hvm, s.appQName, float64(len(items)))

	err = s.storage.PutBatch(items)
	if err == nil {
		for _, i := range items {
			s.cache.Set(key(i.PKey, i.CCols), i.Value)
		}
	}
	return err
}

func (s *cachedAppStorage) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	start := time.Now()
	defer func() {
		s.metrics.IncreaseApp(getSeconds, s.hvm, s.appQName, time.Since(start).Seconds())
	}()
	s.metrics.IncreaseApp(getTotal, s.hvm, s.appQName, 1.0)

	*data = (*data)[0:0]
	*data = s.cache.Get(*data, key(pKey, cCols))
	if len(*data) != 0 {
		s.metrics.IncreaseApp(getCachedTotal, s.hvm, s.appQName, 1.0)
		return true, err
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		s.cache.Set(key(pKey, cCols), *data)
	}
	return
}

func (s *cachedAppStorage) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	start := time.Now()
	defer func() {
		s.metrics.IncreaseApp(getBatchSeconds, s.hvm, s.appQName, time.Since(start).Seconds())
	}()
	s.metrics.IncreaseApp(getBatchTotal, s.hvm, s.appQName, 1.0)
	if !s.getBatchFromCache(pKey, items) {
		return s.getBatchFromStorage(pKey, items)
	}
	return
}

func (s *cachedAppStorage) getBatchFromCache(pKey []byte, items []istorage.GetBatchItem) (ok bool) {
	for i := range items {
		*items[i].Data = s.cache.Get((*items[i].Data)[0:0], key(pKey, items[i].CCols))
		if len(*items[i].Data) == 0 {
			return false
		}
		items[i].Ok = true
	}
	s.metrics.IncreaseApp(getBatchCachedTotal, s.hvm, s.appQName, 1.0)
	return true
}

func (s *cachedAppStorage) getBatchFromStorage(pKey []byte, items []istorage.GetBatchItem) (err error) {
	err = s.storage.GetBatch(pKey, items)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Ok {
			s.cache.Set(key(pKey, item.CCols), *item.Data)
		} else {
			s.cache.Del(key(pKey, item.CCols))
		}
	}
	return err
}

func (s *cachedAppStorage) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	start := time.Now()
	defer func() {
		s.metrics.IncreaseApp(readSeconds, s.hvm, s.appQName, time.Since(start).Seconds())
	}()
	s.metrics.IncreaseApp(readTotal, s.hvm, s.appQName, 1.0)

	return s.storage.Read(ctx, pKey, startCCols, finishCCols, cb)
}

func key(pKey []byte, cCols []byte) []byte {
	return append(pKey, cCols...)
}
