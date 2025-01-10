/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

type cachedAppStorage struct {
	cache    *fastcache.Cache
	storage  istorage.IAppStorage
	vvm      string
	appQName appdef.AppQName
	iTime    coreutils.ITime

	/* metrics */
	mGetSeconds                 *imetrics.MetricValue
	mGetTotal                   *imetrics.MetricValue
	mGetCachedTotal             *imetrics.MetricValue
	mGetBatchSeconds            *imetrics.MetricValue
	mGetBatchTotal              *imetrics.MetricValue
	mGetBatchCachedTotal        *imetrics.MetricValue
	mPutTotal                   *imetrics.MetricValue
	mPutSeconds                 *imetrics.MetricValue
	mPutBatchTotal              *imetrics.MetricValue
	mPutBatchSeconds            *imetrics.MetricValue
	mPutBatchItemsTotal         *imetrics.MetricValue
	mReadTotal                  *imetrics.MetricValue
	mReadSeconds                *imetrics.MetricValue
	mIncreaseIfNotExistsSeconds *imetrics.MetricValue
	mIncreaseIfNotExistsTotal   *imetrics.MetricValue
	mCompareAndSwapSeconds      *imetrics.MetricValue
	mCompareAndSwapTotal        *imetrics.MetricValue
	mCompareAndDeleteSeconds    *imetrics.MetricValue
	mCompareAndDeletepTotal     *imetrics.MetricValue
	mTTLGetSeconds              *imetrics.MetricValue
	mTTLGetTotal                *imetrics.MetricValue
	mTTLReadSeconds             *imetrics.MetricValue
	mTTLReadTotal               *imetrics.MetricValue
}

type implCachingAppStorageProvider struct {
	storageProvider istorage.IAppStorageProvider
	maxBytes        int
	metrics         imetrics.IMetrics
	vvmName         string
	iTime           coreutils.ITime
}

func (asp *implCachingAppStorageProvider) AppStorage(appQName appdef.AppQName) (istorage.IAppStorage, error) {
	nonCachingAppStorage, err := asp.storageProvider.AppStorage(appQName)
	if err != nil {
		return nil, err
	}

	return newCachingAppStorage(
		asp.maxBytes,
		nonCachingAppStorage,
		asp.metrics,
		asp.vvmName,
		appQName,
		asp.iTime,
	), nil
}

func newCachingAppStorage(
	maxBytes int,
	nonCachingAppStorage istorage.IAppStorage,
	metrics imetrics.IMetrics,
	vvm string,
	appQName appdef.AppQName,
	iTime coreutils.ITime,
) istorage.IAppStorage {
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
		iTime:                iTime,
	}
}

//nolint:revive
func (s *cachedAppStorage) InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error) {
	ok, err = s.storage.InsertIfNotExists(pKey, cCols, value, ttlSeconds)
	if err != nil {
		return false, err
	}

	if ok {
		d := dataWithTTL{
			data: value,
			ttl:  s.iTime.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixMilli(),
		}

		dataToCache, err := d.MarshalBinary()
		if err != nil {
			return false, fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
		}
		s.cache.Set(makeKey(pKey, cCols), dataToCache)
	}

	return ok, nil
}

//nolint:revive
func (s *cachedAppStorage) CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error) {
	ok, err = s.storage.CompareAndSwap(pKey, cCols, oldValue, newValue, ttlSeconds)
	if err != nil {
		return false, err
	}

	if ok {
		d := dataWithTTL{
			data: newValue,
			ttl:  s.iTime.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixMilli(),
		}

		dataToCache, err := d.MarshalBinary()
		if err != nil {
			return false, fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
		}
		s.cache.Set(makeKey(pKey, cCols), dataToCache)
	}

	return ok, nil
}

//nolint:revive
func (s *cachedAppStorage) CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error) {
	ok, err = s.storage.CompareAndDelete(pKey, cCols, expectedValue)
	if err != nil {
		return false, fmt.Errorf("storage.CompareAndDelete: %w", err)
	}

	if ok {
		s.cache.Del(makeKey(pKey, cCols))
	}

	return ok, nil
}

//nolint:revive
func (s *cachedAppStorage) TTLGet(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	var found bool
	var key = makeKey(pKey, cCols)

	*data = (*data)[0:0]
	cachedData := make([]byte, 0)
	cachedData, found = s.cache.HasGet(*data, key)

	if found {
		var d dataWithTTL
		if err := d.UnmarshalBinary(cachedData); err != nil {
			return false, fmt.Errorf(fmtErrMsgDataWithTTLUnmarshalBinary, err)
		}

		if isExpired(d.ttl, s.iTime.Now()) {
			s.cache.Del(key)

			return false, nil
		}
		*data = d.data

		return len(*data) != 0, nil
	}

	return s.storage.TTLGet(pKey, cCols, data)
}

//nolint:revive
func (s *cachedAppStorage) TTLRead(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {

	return s.storage.TTLRead(ctx, pKey, startCCols, finishCCols, cb)
}

func (s *cachedAppStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	start := time.Now()
	defer func() {
		s.mPutSeconds.Increase(time.Since(start).Seconds())
	}()
	s.mPutTotal.Increase(1.0)
	err = s.storage.Put(pKey, cCols, value)

	if err == nil {
		data := dataWithTTL{data: value, ttl: maxTTL}
		dataToCache, err := data.MarshalBinary()

		if err != nil {
			return fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
		}
		s.cache.Set(makeKey(pKey, cCols), dataToCache)
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
			data := dataWithTTL{data: i.Value, ttl: maxTTL}
			dataToCache, err := data.MarshalBinary()

			if err != nil {
				return fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
			}
			s.cache.Set(makeKey(i.PKey, i.CCols), dataToCache)
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

	cachedData := make([]byte, 0)
	cachedData, ok = s.cache.HasGet(cachedData, key)

	if ok {
		s.mGetCachedTotal.Increase(1.0)

		if len(cachedData) != 0 {
			var d dataWithTTL
			if err := d.UnmarshalBinary(cachedData); err != nil {
				return false, fmt.Errorf(fmtErrMsgDataWithTTLUnmarshalBinary, err)
			}
			*data = d.data

			return len(*data) != 0, nil
		}
	}

	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, fmt.Errorf("storage.Get: %w", err)
	}

	d := dataWithTTL{ttl: maxTTL}
	if ok {
		d.data = *data
	}

	dataToCache, err := d.MarshalBinary()
	if err != nil {
		return false, fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
	}
	s.cache.Set(key, dataToCache)

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
		cachedData, ok := s.cache.HasGet((*items[i].Data)[0:0], makeKey(pKey, items[i].CCols))
		if !ok {
			return false
		}

		var d dataWithTTL
		if err := d.UnmarshalBinary(cachedData); err != nil {
			return false
		}

		*items[i].Data = d.data
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
		d := dataWithTTL{ttl: maxTTL}
		if item.Ok {
			d.data = *item.Data
		}

		dataToCache, err := d.MarshalBinary()
		if err != nil {
			return fmt.Errorf(fmtErrMsgDataWithTTLMarshalBinary, err)
		}
		s.cache.Set(makeKey(pKey, item.CCols), dataToCache)
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

func isExpired(ttlInMilliseconds int64, now time.Time) bool {
	return !now.Before(time.UnixMilli(ttlInMilliseconds))
}

// dataWithTTL holds some byte data and a TTL in unix milleseconds
type dataWithTTL struct {
	data []byte
	ttl  int64
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (d *dataWithTTL) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write length of Data (8 bytes, big-endian)
	if err := binary.Write(buf, binary.BigEndian, int64(len(d.data))); err != nil {
		return nil, err
	}

	// Write the Data bytes
	if _, err := buf.Write(d.data); err != nil {
		return nil, err
	}

	// Write TTL (8 bytes, big-endian)
	if err := binary.Write(buf, binary.BigEndian, d.ttl); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (d *dataWithTTL) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	// Read length of Data
	var dataLen uint64
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return err
	}

	// Read Data bytes
	d.data = make([]byte, dataLen)
	if _, err := io.ReadFull(buf, d.data); err != nil {
		return err
	}

	// Read TTL
	if err := binary.Read(buf, binary.BigEndian, &d.ttl); err != nil {
		return err
	}

	return nil
}
