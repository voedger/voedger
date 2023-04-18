/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	istructs "github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

const (
	testCacheSize = 1000
)

func TestBasicUsage(t *testing.T) {
	t.Run("Should put record to storage and cache after that use cache for read", func(t *testing.T) {
		require := require.New(t)
		times := 0
		ts := &testStorage{
			put: func(pKey []byte, cCols []byte, value []byte) (err error) {
				require.Equal([]byte("UK"), pKey)
				require.Equal([]byte("Article"), cCols)
				require.Equal([]byte("Cola"), value)
				return err
			},
			get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
				times++
				return ok, err
			},
		}
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		_ = storage.Put([]byte("UK"), []byte("Article"), []byte("Cola"))

		_, _ = storage.Get([]byte("UK"), []byte("Article"), &[]byte{})
		_, _ = storage.Get([]byte("UK"), []byte("Article"), &[]byte{})

		require.Equal(0, times)
	})
	t.Run("Should put records to storage and cache after that use cache for read", func(t *testing.T) {
		require := require.New(t)
		times := 0
		ts := &testStorage{
			putBatch: func(items []istorage.BatchItem) (err error) {
				require.Len(items, 1)
				return err
			},
			get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
				times++
				return ok, err
			},
		}
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		_ = storage.PutBatch([]istorage.BatchItem{{
			PKey:  []byte("UK"),
			CCols: []byte("Article"),
			Value: []byte("Cola"),
		}})

		_, _ = storage.Get([]byte("UK"), []byte("Article"), &[]byte{})
		_, _ = storage.Get([]byte("UK"), []byte("Article"), &[]byte{})

		require.Equal(0, times)
	})
	t.Run("Should get record from storage after that put it to cache and use cache for read", func(t *testing.T) {
		require := require.New(t)
		times := 0
		ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
			times++
			*data = append(*data, []byte("Cola")...)
			return true, nil
		}}
		data := make([]byte, 0, 100)
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		_, _ = storage.Get([]byte("UK"), []byte("Article"), &data)
		require.Equal([]byte("Cola"), data)

		data = make([]byte, 0, 100)
		_, _ = storage.Get([]byte("UK"), []byte("Article"), &data)
		require.Equal([]byte("Cola"), data)

		require.Equal(1, times)
	})
	t.Run("Should get batch from cache when at least one item from batch is not in the cache then complete batch re-read from underlying storage after that put batch to cache and use cache for read", func(t *testing.T) {
		require := require.New(t)
		ts := &testStorage{
			put: func(pKey []byte, cCols []byte, value []byte) (err error) { return err },
			getBatch: func(pKey []byte, items []istorage.GetBatchItem) (err error) {
				*items[0].Data = append((*items[0].Data)[0:0], []byte("Cola")...)
				items[0].Ok = true
				items[1].Ok = false
				*items[2].Data = append((*items[2].Data)[0:0], []byte("Napkin")...)
				items[2].Ok = true
				return err
			},
		}
		data := make([]byte, 0, 100)
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		_ = storage.Put([]byte("NL"), []byte("Beverage"), []byte("Cola"))
		items := []istorage.GetBatchItem{
			{
				CCols: []byte("Beverage"),
				Data:  &[]byte{},
			},
			{
				CCols: []byte("Food"),
				Data:  &[]byte{},
			},
			{
				CCols: []byte("Misc"),
				Data:  &[]byte{},
			},
		}

		_ = storage.GetBatch([]byte("NL"), items)

		require.Equal([]byte("Cola"), *items[0].Data)
		require.Empty(items[1].Data)
		require.Equal([]byte("Napkin"), *items[2].Data)

		_, _ = storage.Get([]byte("NL"), items[0].CCols, &data)
		require.Equal([]byte("Cola"), data)
		_, _ = storage.Get([]byte("NL"), items[2].CCols, &data)
		require.Equal([]byte("Napkin"), data)
	})

	t.Run("error on app storage get error", func(t *testing.T) {
		testErr := errors.New("test error")
		tsp := &testStorageProvider{appStorageGetError: testErr}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.ErrorIs(t, err, testErr)
		require.Nil(t, storage)
	})
}

func TestAppStorage_Put(t *testing.T) {
	require := require.New(t)
	testErr := errors.New("test error")
	ts := &testStorage{
		put: func(pKey []byte, cCols []byte, value []byte) (err error) {
			return testErr
		},
		get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
			return false, err
		},
	}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	err = storage.Put([]byte("UK"), []byte("Article"), []byte("Cola"))

	require.ErrorIs(err, testErr)

	ok, err := storage.Get([]byte("UK"), []byte("Article"), &[]byte{})

	require.False(ok)
	require.Nil(err)
}

func TestAppStorage_PutBatch(t *testing.T) {
	require := require.New(t)
	testErr := errors.New("test error")
	ts := &testStorage{
		putBatch: func(items []istorage.BatchItem) (err error) {
			return testErr
		},
		get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
			return false, err
		},
	}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	err = storage.PutBatch([]istorage.BatchItem{{
		PKey:  []byte("UK"),
		CCols: []byte("Article"),
		Value: []byte("Cola"),
	}})

	require.ErrorIs(err, testErr)

	ok, err := storage.Get([]byte("UK"), []byte("Article"), &[]byte{})

	require.False(ok)
	require.Nil(err)
}

func TestAppStorage_Get(t *testing.T) {
	require := require.New(t)
	testErr := errors.New("test error")
	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		return false, testErr
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	ok, err := storage.Get([]byte("UK"), []byte("Article"), &[]byte{})

	require.False(ok)
	require.ErrorIs(err, testErr)
}

func TestAppStorage_GetBatch(t *testing.T) {
	t.Run("Should handle error", func(t *testing.T) {
		require := require.New(t)
		testErr := errors.New("test error")
		ts := &testStorage{getBatch: func(pKey []byte, items []istorage.GetBatchItem) (err error) {
			return testErr
		}}
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		err = storage.GetBatch([]byte("UK"), []istorage.GetBatchItem{{
			CCols: []byte("Article"),
			Data:  &[]byte{},
		}})

		require.ErrorIs(err, testErr)
	})
	t.Run("Should remove item from cache when it was not found from underlying storage", func(t *testing.T) {
		require := require.New(t)
		ts := &testStorage{
			get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
				return false, err
			},
			put: func(pKey []byte, cCols []byte, value []byte) (err error) {
				return err
			},
			getBatch: func(pKey []byte, items []istorage.GetBatchItem) (err error) {
				items[0].Ok = false
				*items[1].Data = append((*items[1].Data)[0:0], []byte("Burger ver.1.1")...)
				items[1].Ok = true
				*items[2].Data = append((*items[2].Data)[0:0], []byte("Napkin ver.1.0")...)
				items[2].Ok = true
				return err
			}}
		tsp := &testStorageProvider{storage: ts}
		cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "hvm")
		storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		_ = storage.Put([]byte("UK"), []byte("Beverage"), []byte("Cola ver.1.0"))
		_ = storage.Put([]byte("UK"), []byte("Food"), []byte("Burger ver.1.0"))

		_ = storage.GetBatch([]byte("UK"), []istorage.GetBatchItem{
			{
				CCols: []byte("Beverage"),
				Data:  &[]byte{},
			},
			{
				CCols: []byte("Food"),
				Data:  &[]byte{},
			},
			{
				CCols: []byte("Misc"),
				Data:  &[]byte{},
			},
		})

		data := make([]byte, 0, 100)

		ok, _ := storage.Get([]byte("UK"), []byte("Beverage"), &data)
		require.False(ok)
		_, _ = storage.Get([]byte("UK"), []byte("Food"), &data)
		require.Equal([]byte("Burger ver.1.1"), data)
		_, _ = storage.Get([]byte("UK"), []byte("Misc"), &data)
		require.Equal([]byte("Napkin ver.1.0"), data)
	})
}

func TestTechnologyCompatibilityKit(t *testing.T) {
	asf := istorage.ProvideMem()
	asp := istorageimpl.Provide(asf)
	cachingStorageProvider := Provide(testCacheSize, asp, imetrics.Provide(), "hvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(t, err)
	istorage.TechnologyCompatibilityKit_Storage(t, storage)
}

type testStorageProvider struct {
	storage            *testStorage
	appStorageGetError error
}

func (sp *testStorageProvider) AppStorage(istructs.AppQName) (istorage.IAppStorage, error) {
	if sp.appStorageGetError != nil {
		return nil, sp.appStorageGetError
	}
	return sp.storage, nil
}

type testStorage struct {
	put      func(pKey []byte, cCols []byte, value []byte) (err error)
	putBatch func(items []istorage.BatchItem) (err error)
	get      func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)
	getBatch func(pKey []byte, items []istorage.GetBatchItem) (err error)
}

func (s *testStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	return s.put(pKey, cCols, value)
}

func (s *testStorage) PutBatch(items []istorage.BatchItem) (err error) {
	return s.putBatch(items)
}

func (s *testStorage) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	return s.get(pKey, cCols, data)
}

func (s *testStorage) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	return s.getBatch(pKey, items)
}

func (s *testStorage) Read(context.Context, []byte, []byte, []byte, istorage.ReadCallback) (err error) {
	return err
}
