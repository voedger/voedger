/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 * @author: Maxim Geraskin (refactoring)
 */

package bbolt

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
)

type appStorageFactory struct {
	bboltParams ParamsType
	iTime       coreutils.ITime
}

func (p *appStorageFactory) AppStorage(appName istorage.SafeAppName) (s istorage.IAppStorage, err error) {
	dbName := filepath.Join(p.bboltParams.DBDir, appName.String()+".db")
	exists, err := coreutils.Exists(dbName)
	if err != nil {
		// notest
		return nil, err
	}
	if !exists {
		return nil, istorage.ErrStorageDoesNotExist
	}
	db, err := bolt.Open(dbName, coreutils.FileMode_rw_rw_rw_, bolt.DefaultOptions)
	if err != nil {
		// notest
		return nil, err
	}
	return &appStorageType{db: db, iTime: p.iTime}, nil
}

func (p *appStorageFactory) Init(appName istorage.SafeAppName) error {
	dbName := filepath.Join(p.bboltParams.DBDir, appName.String()+".db")
	exists, err := coreutils.Exists(dbName)
	if err != nil {
		// notest
		return err
	}
	if exists {
		return istorage.ErrStorageAlreadyExists
	}
	if err = os.MkdirAll(p.bboltParams.DBDir, coreutils.FileMode_rwxrwxrwx); err != nil {
		// notest
		return err
	}
	db, err := bolt.Open(dbName, coreutils.FileMode_rw_rw_rw_, bolt.DefaultOptions)
	if err != nil {
		// notest
		return err
	}
	return db.Close()
}

// bolt cannot use empty keys so we declare nullKey
var nullKey = []byte{0}

// if the key is empty or equal to nil, then convert it to nullKey
func safeKey(value []byte) []byte {
	if len(value) == 0 {
		return nullKey
	}
	return value
}

// if the key is nullKey, then convert it to nil
func unSafeKey(value []byte) []byte {
	if len(value) == 0 || (len(value) == 1 && value[0] == 0) {
		return nil
	}
	return value
}

// implemetation for istorage.IAppStorage.
type appStorageType struct {
	db             *bolt.DB
	iTime          coreutils.ITime
	chKeysToRemove chan []byte
}

// Vision about TTL implementation:
// TTL-keys will be stored in global bucket called "ttl_keys" (name must be configured via constant).
// Inside ttl_keys bucket, there are buckets for year, inside year will be months then inside each month will be buckets for days.
// Each day stores buckets of cleanup hours. Each cleanup hour stores buckets for keys to be deleted.
// year, month, day and cleanup hour will be calculated via cleanup time.
// Cleanup time will be calculated as current time + ttlSeconds rounded to next hour.
// Cleanup tree:
// ttl_keys -> 2025, 2026
// 2025 -> 1, 2
// 1 -> 0, 1, 2, 15
// 0 -> key1, key2, key3
// 1 -> key4, key5, key6
// 15 -> key4, key5, key6
// keys are leaves and cleanup nodes (year, months, days, hours are branches)
// each ttl key is stored in ttl_keys bucket for direct read purposes.
// The cleanup goroutine must be started  for each appStorageType instance.
// The goroutine calculate next cleanup hour and sleep until that time.
// When the time comes, it will read all expired keys and removed them directly and from cleanup tree then sleeps till next cleanup hour.
// If the goroutine finds expired nodes in cleanup (for example: it is the august of the 2025, but there is july node in 2025 parent then),
// then all expired keys must be removed, then expired branches.
// So the cleanup goroutine starts each hour.

//nolint:revive
func (s *appStorageType) InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error) {
	//TODO implement me
	panic("implement me")
}

//nolint:revive
func (s *appStorageType) CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error) {
	//TODO implement me
	panic("implement me")
}

//nolint:revive
func (s *appStorageType) CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error) {
	//TODO implement me
	panic("implement me")
}

//nolint:revive
func (s *appStorageType) TTLGet(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	//TODO implement me
	panic("implement me")
}

//nolint:revive
func (s *appStorageType) TTLRead(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	//TODO implement me
	panic("implement me")
}

func getCleanupTime(now time.Time) time.Time {
	// get next hour
	nextHour := now.Add(time.Hour)
	return time.Date(nextHour.Year(), nextHour.Month(), nextHour.Day(), nextHour.Hour(), 0, 0, 0, time.UTC)
}

// cleanup goroutine
func (s *appStorageType) cleanup(ctx context.Context) {
	for {
		now := s.iTime.Now()

		nextCleanupTime := getCleanupTime(now)
		timer := time.NewTimer(nextCleanupTime.Sub(now))

		select {
		case <-timer.C:
			cleanupTime := s.iTime.Now()
			_ = s.removeExpiredBranchNodesFromCleanupTree(cleanupTime)
		case key := <-s.chKeysToRemove:
			_ = s.removeKeyFromCleanupTree(key)
		case <-ctx.Done():
			return
		}
	}
}

func (s *appStorageType) removeKeyFromCleanupTree(key []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		ttlBucket := tx.Bucket([]byte(ttlBucketName))
		if ttlBucket == nil {
			return nil
		}

		value := ttlBucket.Get(key)
		if value == nil {
			return nil
		}

		if err := ttlBucket.Delete(key); err != nil {
			return err
		}

		var d coreutils.DataWithExpiration
		d.Read(value)

		cleanupTime := getCleanupTime(time.UnixMilli(d.ExpireAt))
		yearBucketKey, monthBucketKey, dayBucketKey, hourBucketKey := getCleanupNodes(cleanupTime)

		yearBucket := ttlBucket.Bucket(yearBucketKey)
		if yearBucket == nil {
			return nil
		}

		monthBucket := yearBucket.Bucket(monthBucketKey)
		if monthBucket == nil {
			return nil
		}

		dayBucket := monthBucket.Bucket(dayBucketKey)
		if dayBucket == nil {
			return nil
		}

		hourBucket := dayBucket.Bucket(hourBucketKey)
		if hourBucket == nil {
			return nil
		}

		if err := hourBucket.Delete(key); err != nil {
			return err
		}

		// remove empty branch node buckets from tree
		countOfKeys, err := countOfKeysInBucket(hourBucket)
		if err != nil {
			return err
		}

		if countOfKeys == 0 {
			if err := dayBucket.Delete(hourBucketKey); err != nil {
				return err
			}

			countOfKeys, err := countOfKeysInBucket(dayBucket)
			if err != nil {
				return err
			}

			if countOfKeys == 0 {
				if err := monthBucket.Delete(dayBucketKey); err != nil {
					return err
				}

				countOfKeys, err := countOfKeysInBucket(monthBucket)
				if err != nil {
					return err
				}

				if countOfKeys == 0 {
					if err := yearBucket.Delete(monthBucketKey); err != nil {
						return err
					}

					countOfKeys, err := countOfKeysInBucket(yearBucket)
					if err != nil {
						return err
					}

					if countOfKeys == 0 {
						if err := ttlBucket.Delete(yearBucketKey); err != nil {
							return err
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func removeNodeBucketFromCleanupTree(rootBucket, parentNodeBucket *bolt.Bucket, nodeBucketKey []byte) error {
	var nodeBucket *bolt.Bucket
	if parentNodeBucket != nil {
		nodeBucket = parentNodeBucket.Bucket(nodeBucketKey)
	} else {
		nodeBucket = rootBucket.Bucket(nodeBucketKey)
	}

	if nodeBucket == nil {
		return nil
	}

	childBucketCount := 0
	err := nodeBucket.ForEachBucket(func(childBucketKey []byte) error {
		childBucketCount++
		err := removeNodeBucketFromCleanupTree(rootBucket, nodeBucket, childBucketKey)
		if err != nil {
			return err
		}

		return nodeBucket.DeleteBucket(childBucketKey)
	})
	if err != nil {
		return err
	}

	// we deal with last branch node, then we must read all leaf nodes (keys) and remove them
	if childBucketCount == 0 {
		keys := make([][]byte, 0, nodeBucket.Stats().KeyN)

		err := nodeBucket.ForEach(func(k, _ []byte) error {
			keys = append(keys, k)

			return nil
		})
		if err != nil {
			return err
		}

		for _, key := range keys {
			if err := rootBucket.Delete(key); err != nil {
				return err
			}

			if err := nodeBucket.Delete(key); err != nil {
				return err
			}
		}
	}

	return rootBucket.DeleteBucket(nodeBucketKey)
}

func (s *appStorageType) removeExpiredBranchNodesFromCleanupTree(cleanupTime time.Time) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		ttlBucket := tx.Bucket([]byte(ttlBucketName))
		if ttlBucket == nil {
			return nil
		}

		yearBucketKey, monthBucketKey, dayBucketKey, _ := getCleanupNodes(cleanupTime)

		previousYear := cleanupTime.Year() - 1
		previousYearBucketKey := []byte(strconv.Itoa(previousYear))

		if previousYearBucket := ttlBucket.Bucket(previousYearBucketKey); previousYearBucket != nil {
			if err := removeNodeBucketFromCleanupTree(ttlBucket, nil, previousYearBucketKey); err != nil {
				return err
			}
		}

		previousMonth := cleanupTime.Month() - 1
		if previousMonth == 0 {
			previousMonth = 12
		}

		yearBucket := ttlBucket.Bucket(yearBucketKey)
		if yearBucket == nil {
			return nil
		}

		previousMonthKey := []byte(strconv.Itoa(int(previousMonth)))
		if previousMonthBucket := yearBucket.Bucket(previousMonthKey); previousMonthBucket != nil {
			if err := removeNodeBucketFromCleanupTree(ttlBucket, yearBucket, previousMonthKey); err != nil {
				return err
			}
		}

		monthBucket := yearBucket.Bucket(monthBucketKey)
		if monthBucket == nil {
			return nil
		}

		previousDayTime := cleanupTime.AddDate(0, 0, -1)
		previousDay := previousDayTime.Day()

		previousDayKey := []byte(strconv.Itoa(previousDay))
		if previousDayBucket := monthBucket.Bucket(previousDayKey); previousDayBucket != nil {
			if err := removeNodeBucketFromCleanupTree(ttlBucket, monthBucket, previousDayKey); err != nil {
				return err
			}
		}

		dayBucket := monthBucket.Bucket(dayBucketKey)
		if dayBucket == nil {
			return nil
		}

		previousHour := cleanupTime.Hour() - 1
		if previousHour == -1 {
			previousHour = 23
		}

		previousHourKey := []byte(strconv.Itoa(previousHour))
		if previousHourBucket := dayBucket.Bucket(previousHourKey); previousHourBucket != nil {
			if err := removeNodeBucketFromCleanupTree(ttlBucket, dayBucket, previousHourKey); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *appStorageType) putKeyInCleanupTree(key, value []byte, ttlSeconds int64) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		ttlBucket, e := tx.CreateBucketIfNotExists([]byte(ttlBucketName))
		if e != nil {
			return e
		}

		expireAt := s.iTime.Now().Add(time.Duration(ttlSeconds) * time.Second)
		d := coreutils.DataWithExpiration{
			Data:     value,
			ExpireAt: expireAt.UnixMilli(),
		}

		cleanupTime := getCleanupTime(expireAt)
		yearBucketKey, monthBucketKey, dayBucketKey, hourBucketKey := getCleanupNodes(cleanupTime)

		yearBucket, err := ttlBucket.CreateBucketIfNotExists(yearBucketKey)
		if err != nil {
			return err
		}

		monthBucket, err := yearBucket.CreateBucketIfNotExists(monthBucketKey)
		if err != nil {
			return err
		}

		dayBucket, err := monthBucket.CreateBucketIfNotExists(dayBucketKey)
		if err != nil {
			return err
		}

		hourBucket, err := dayBucket.CreateBucketIfNotExists(hourBucketKey)
		if err != nil {
			return err
		}

		if err := hourBucket.Put(key, []byte{0}); err != nil {
			return err
		}

		if err := ttlBucket.Put(key, d.ToBytes()); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func getCleanupNodes(cleanupTime time.Time) (year, month, day, hour []byte) {
	year = []byte(strconv.Itoa(cleanupTime.Year()))
	month = []byte(strconv.Itoa(int(cleanupTime.Month())))
	day = []byte(strconv.Itoa(cleanupTime.Day()))
	hour = []byte(strconv.Itoa(cleanupTime.Hour()))

	return
}

// istorage.IAppStorage.Put(pKey []byte, cCols []byte, value []byte) (err error)
func (s *appStorageType) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(pKey)
		if e != nil {
			// notest
			return e
		}
		return b.Put(safeKey(cCols), unSafeKey(value))
	})
	return err
}

// istorage.IAppStorage.PutBatch(items []BatchItem) (err error)
func (s *appStorageType) PutBatch(items []istorage.BatchItem) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {

		for i := 0; i < len(items); i++ {

			PKey := items[i].PKey
			b, e := tx.CreateBucketIfNotExists(PKey)
			if e != nil {
				// notest
				return e
			}

			e = b.Put(safeKey(items[i].CCols), items[i].Value)
			if e != nil {
				return e
			}
		}

		return nil
	})

	return err
}

// istorage.IAppStorage.Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)
func (s *appStorageType) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	*data = (*data)[0:0]

	err = s.db.View(func(tx *bolt.Tx) error {
		ok = false
		bucket := tx.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		v := bucket.Get(safeKey(cCols))
		if v == nil {
			return nil
		}
		*data = append(*data, v...)
		ok = true
		return nil
	})

	return ok, err
}

// istorage.IAppStorage.Read(ctx context.Context, pKey []byte, startCCols []byte, finishCCols []byte, cb ReadCallback) (err error)
func (s *appStorageType) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {

	if (len(startCCols) > 0) && (len(finishCCols) > 0) && (bytes.Compare(startCCols, finishCCols) >= 0) {
		return nil // absurd range
	}

	err = s.db.View(func(tx *bolt.Tx) error {

		startCCols = unSafeKey(startCCols)
		finishCCols = unSafeKey(finishCCols)

		bucket := tx.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		var (
			k []byte
			v []byte
		)

		cr := bucket.Cursor()
		if startCCols == nil {
			k, v = cr.First()
		} else {
			k, v = cr.Seek(startCCols)
		}

		var e error

		for (k != nil) && (finishCCols == nil || string(k) <= string(finishCCols)) {

			if ctx.Err() != nil {
				return nil
			}

			if cb != nil {
				e = cb(unSafeKey(k), unSafeKey(v))
				if e != nil {
					return e
				}
			}
			k, v = cr.Next()
		}

		return nil
	})

	return err
}

// istorage.IAppStorage.GetBatch(pKey []byte, items []GetBatchItem) (err error)
func (s *appStorageType) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	err = s.db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket(pKey)
		if bucket == nil {
			for i := 0; i < len(items); i++ {
				items[i].Ok = false
			}
			return nil
		}
		for i := 0; i < len(items); i++ {
			v := bucket.Get(safeKey(items[i].CCols))
			items[i].Ok = v != nil
			*items[i].Data = append((*items[i].Data)[0:0], v...)
		}
		return nil
	})

	return err
}

func (p *appStorageFactory) Time() coreutils.ITime {
	return nil
}

func countOfKeysInBucket(bucket *bolt.Bucket) (count int, err error) {
	if bucket == nil {
		return 0, nil
	}

	count = 0
	err = bucket.ForEach(func(k, v []byte) error {
		count++
		return nil
	})

	return
}
