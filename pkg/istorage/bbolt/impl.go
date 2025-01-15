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
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"os"
	"path/filepath"
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

//nolint:revive
func (s *appStorageType) InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error) {
	found := false

	key := makeKey(pKey, cCols)
	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		d := dataBucket.Get(key)
		if d == nil {
			return nil
		}

		expireAt := time.UnixMilli(int64(binary.BigEndian.Uint64(d[len(d)-utils.Uint64Size:])))
		if !expireAt.After(s.iTime.Now()) {
			s.chKeysToRemove <- key

			return nil
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	if found {
		return false, nil
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		return s.putValue(tx, key, value, ttlSeconds)
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

//nolint:revive
func (s *appStorageType) CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error) {
	found := false

	key := makeKey(pKey, cCols)
	found, err = s.findValue(key, oldValue)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		return s.putValue(tx, key, newValue, ttlSeconds)
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

//nolint:revive
func (s *appStorageType) CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error) {
	found := false

	key := makeKey(pKey, cCols)
	found, err = s.findValue(key, expectedValue)
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

		if err := dataBucket.Delete(key); err != nil {
			return err
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
	key := makeKey(pKey, cCols)
	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		d := dataBucket.Get(key)
		if d == nil {
			return nil
		}

		expireAt := time.UnixMilli(int64(binary.BigEndian.Uint64(d[len(d)-utils.Uint64Size:])))
		if !expireAt.After(s.iTime.Now()) {
			s.chKeysToRemove <- key

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
	//TODO implement me
	panic("implement me")
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

func (s *appStorageType) findValue(key, value []byte) (found bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		if dataBucket == nil {
			return ErrDataBucketNotFound
		}

		d := dataBucket.Get(key)
		if d == nil {
			return nil
		}

		expireAt := time.UnixMilli(int64(binary.BigEndian.Uint64(d[len(d)-utils.Uint64Size:])))
		if !expireAt.After(s.iTime.Now()) {
			s.chKeysToRemove <- key

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

func (s *appStorageType) putValue(tx *bolt.Tx, key, value []byte, ttlSeconds int) error {
	dataBucket := tx.Bucket([]byte(dataBucketName))
	if dataBucket == nil {
		return ErrDataBucketNotFound
	}

	d := coreutils.DataWithExpiration{
		Data: value,
	}
	if ttlSeconds > 0 {
		d.ExpireAt = s.iTime.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixMilli()
	}

	if err := dataBucket.Put(key, d.ToBytes()); err != nil {
		return err
	}

	if ttlSeconds > 0 {
		ttlBucket := tx.Bucket([]byte(ttlBucketName))
		if ttlBucket == nil {
			return ErrTTLBucketNotFound
		}

		if err := ttlBucket.Put(makeTtlKey(key, d.ExpireAt), nil); err != nil {
			return err
		}
	}

	return nil
}

func (p *appStorageFactory) Time() coreutils.ITime {
	return nil
}

func makeKey(pKey []byte, cCols []byte) []byte {
	key := make([]byte, 0, len(pKey)+len(cCols))
	key = append(key, pKey...)
	key = append(key, cCols...)

	return key
}

func makeTtlKey(key []byte, expireAt int64) []byte {
	ttlKey := make([]byte, 0, len(key)+utils.Uint64Size)
	binary.BigEndian.AppendUint64(ttlKey, uint64(expireAt)) // nolint G115
	key = append(key, key...)

	return key
}
