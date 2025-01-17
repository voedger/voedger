/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 * @author: Maxim Geraskin (refactoring)
 */

package bbolt

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
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
	// initialize the database
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucket([]byte(dataBucketName)); err != nil {
			return err
		}
		if _, err = tx.CreateBucket([]byte(ttlBucketName)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return db.Close()
}

func (p *appStorageFactory) Time() coreutils.ITime {
	return p.iTime
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

//nolint:revive
func (s *appStorageType) InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error) {
	found := false

	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		d := bucket.Get(safeKey(cCols))
		if d == nil {
			return nil
		}

		if s.isExpired(d) {
			//s.chKeysToRemove <- makeKey(pKey, cCols)

			return nil
		}

		found = true

		return nil
	})
	if err != nil {
		return false, err
	}

	if found {
		return false, nil
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		return s.putValue(tx, pKey, cCols, value, ttlSeconds)
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

//nolint:revive
func (s *appStorageType) CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error) {
	found := false

	found, err = s.findValue(pKey, cCols, oldValue)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		return s.putValue(tx, pKey, cCols, newValue, ttlSeconds)
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

//nolint:revive
func (s *appStorageType) CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error) {
	found, err := s.findValue(pKey, cCols, expectedValue)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		if err := bucket.Delete(cCols); err != nil {
			return err
		}

		if bucket.Stats().KeyN == 0 {
			if err := dataBucket.DeleteBucket(pKey); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

//nolint:revive
func (s *appStorageType) TTLGet(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		d := bucket.Get(cCols)
		if d == nil {
			return nil
		}

		if s.isExpired(d) {
			//s.chKeysToRemove <- makeKey(pKey, cCols)

			return nil
		}

		*data = (*data)[:0]
		*data = d[:len(d)-utils.Uint64Size]
		ok = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return
}

//nolint:revive
func (s *appStorageType) TTLRead(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	return s.read(ctx, pKey, startCCols, finishCCols, cb, true)
}

// istorage.IAppStorage.Put(pKey []byte, cCols []byte, value []byte) (err error)
func (s *appStorageType) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	return s.db.Update(func(tx *bolt.Tx) error {
		dataBucket, err := tx.CreateBucketIfNotExists([]byte(dataBucketName))
		if err != nil {
			return err
		}
		d := coreutils.DataWithExpiration{Data: value}

		bucket, err := dataBucket.CreateBucketIfNotExists(pKey)
		if err != nil {
			return err
		}

		return bucket.Put(safeKey(cCols), d.ToBytes())
	})
}

// istorage.IAppStorage.PutBatch(items []BatchItem) (err error)
func (s *appStorageType) PutBatch(items []istorage.BatchItem) (err error) {
	return s.db.Update(func(tx *bolt.Tx) error {
		dataBucket, err := tx.CreateBucketIfNotExists([]byte(dataBucketName))
		if err != nil {
			return err
		}

		for i := 0; i < len(items); i++ {
			bucket, err := dataBucket.CreateBucketIfNotExists(items[i].PKey)
			if err != nil {
				return err
			}

			d := coreutils.DataWithExpiration{Data: items[i].Value}
			if err := bucket.Put(items[i].CCols, d.ToBytes()); err != nil {
				return err
			}
		}

		return nil
	})
}

// istorage.IAppStorage.Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)
func (s *appStorageType) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	*data = (*data)[0:0]

	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		v := bucket.Get(safeKey(cCols))
		if v == nil {
			return nil
		}

		var d coreutils.DataWithExpiration

		d.Read(v)
		*data = append(*data, d.Data...)
		ok = true

		return nil
	})

	return ok, err
}

// istorage.IAppStorage.Read(ctx context.Context, pKey []byte, startCCols []byte, finishCCols []byte, cb ReadCallback) (err error)
func (s *appStorageType) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	return s.read(ctx, pKey, startCCols, finishCCols, cb, false)
}

// istorage.IAppStorage.GetBatch(pKey []byte, items []GetBatchItem) (err error)
func (s *appStorageType) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	return s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			for i := 0; i < len(items); i++ {
				items[i].Ok = false
			}
			return nil
		}

		var d coreutils.DataWithExpiration
		for i := 0; i < len(items); i++ {
			v := bucket.Get(safeKey(items[i].CCols))
			if v == nil {
				items[i].Ok = false
				*items[i].Data = append((*items[i].Data)[0:0], v...)
				continue
			}

			d.Read(v)

			items[i].Ok = d.Data != nil
			*items[i].Data = append((*items[i].Data)[0:0], d.Data...)
		}

		return nil
	})
}

func (s *appStorageType) read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback, checkTtl bool) (err error) {
	if (len(startCCols) > 0) && (len(finishCCols) > 0) && (bytes.Compare(startCCols, finishCCols) >= 0) {
		return nil // absurd range
	}

	return s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		startCCols = unSafeKey(startCCols)
		finishCCols = unSafeKey(finishCCols)

		var (
			k []byte
			v []byte
		)

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		cr := bucket.Cursor()
		if startCCols == nil {
			k, v = cr.First()
		} else {
			k, v = cr.Seek(safeKey(startCCols))
		}

		for (k != nil) && (finishCCols == nil || string(k) <= string(finishCCols)) {

			if ctx.Err() != nil {
				return nil
			}

			if checkTtl && s.isExpired(v) {
				//s.chKeysToRemove <- makeKey(pKey, k)
				k, v = cr.Next()
				continue
			}

			if cb != nil {
				if err != nil {
					return err
				}

				if err := cb(unSafeKey(k), v[:len(v)-utils.Uint64Size]); err != nil {
					return err
				}
			}
			k, v = cr.Next()
		}

		return nil
	})
}

func (s *appStorageType) findValue(pKey, cCols, value []byte) (found bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		bucket := dataBucket.Bucket(pKey)
		if bucket == nil {
			return nil
		}

		d := bucket.Get(cCols)
		if d == nil {
			return nil
		}

		if s.isExpired(d) {
			//s.chKeysToRemove <- makeKey(pKey, cCols)

			return nil
		}

		if bytes.Equal(d[:len(d)-utils.Uint64Size], value) {
			found = true
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return found, nil
}

func (s *appStorageType) putValue(tx *bolt.Tx, pKey, cCols, value []byte, ttlSeconds int) error {
	dataBucket, err := tx.CreateBucketIfNotExists([]byte(dataBucketName))
	if err != nil {
		return err
	}

	bucket, err := dataBucket.CreateBucketIfNotExists(pKey)
	if err != nil {
		return err
	}

	d := coreutils.DataWithExpiration{Data: value}
	if ttlSeconds > 0 {
		d.ExpireAt = s.iTime.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixMilli()
	}

	if err := bucket.Put(safeKey(cCols), d.ToBytes()); err != nil {
		return err
	}

	if ttlSeconds > 0 {
		ttlBucket := tx.Bucket([]byte(ttlBucketName))
		if ttlBucket == nil {
			return ErrTTLBucketNotFound
		}

		if err := ttlBucket.Put(makeTtlKey(makeKey(pKey, cCols), d.ExpireAt), nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *appStorageType) isExpired(value []byte) bool {
	expireAtMillisec := int64(binary.BigEndian.Uint64(value[len(value)-utils.Uint64Size:]))
	if expireAtMillisec == 0 {
		return false
	}

	expireAt := time.UnixMilli(expireAtMillisec)
	if expireAt.After(s.iTime.Now()) {
		return false
	}

	return true
}

// makeKey creates a key from the primary key and clustering columns
// key = len(pKey) + pKey + cCols
func makeKey(pKey []byte, cCols []byte) []byte {
	totalLength := utils.Uint64Size + len(pKey) + len(cCols)
	key := make([]byte, totalLength)
	binary.BigEndian.PutUint64(key[:utils.Uint64Size], uint64(len(pKey)))
	copy(key[utils.Uint64Size:utils.Uint64Size+len(pKey)], pKey)
	copy(key[utils.Uint64Size+len(pKey):], cCols)

	return key
}

// extractCCols extracts the clustering columns (cCols) from the composite key.
// It assumes the key is in the format: len(pKey) + pKey + cCols.
func extractCCols(key []byte) ([]byte, error) {
	if len(key) < utils.Uint64Size { // 8 bytes for len(pKey)
		return nil, fmt.Errorf("invalid key: too short")
	}
	pKeyLength := binary.BigEndian.Uint64(key[:utils.Uint64Size])

	cColsOffset := utils.Uint64Size + pKeyLength
	if len(key) < int(cColsOffset) {
		return nil, fmt.Errorf("invalid key: missing cCols data")
	}
	cCols := key[cColsOffset:]

	return cCols, nil
}

// makeTtlKey creates a key for TTL bucket
func makeTtlKey(key []byte, expireAt int64) []byte {
	ttlKey := make([]byte, 0, len(key)+utils.Uint64Size)
	binary.BigEndian.AppendUint64(ttlKey, uint64(expireAt)) // nolint G115
	key = append(key, key...)

	return key
}
