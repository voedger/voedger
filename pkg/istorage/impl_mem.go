/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istorage

import (
	"bytes"
	"context"
	"sort"
	"sync"
)

type appStorageFactory struct {
	storages map[string]map[string]map[string][]byte
}

func (s *appStorageFactory) AppStorage(appName SafeAppName) (IAppStorage, error) {
	storage, ok := s.storages[appName.String()]
	if !ok {
		return nil, ErrStorageDoesNotExist
	}
	return &appStorage{storage: storage}, nil
}

func (s *appStorageFactory) Init(appName SafeAppName) error {
	if _, ok := s.storages[appName.String()]; ok {
		return ErrStorageAlreadyExists
	}
	s.storages[appName.String()] = map[string]map[string][]byte{}
	return nil
}

type appStorage struct {
	storage map[string]map[string][]byte
	lock    sync.RWMutex
}

func (s *appStorage) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	p := s.storage[string(pKey)]
	if p == nil {
		p = make(map[string][]byte)
		s.storage[string(pKey)] = p
	}
	p[string(cCols)] = copySlice(value)
	return
}

func (s *appStorage) PutBatch(items []BatchItem) (err error) {
	for _, item := range items {
		err = s.Put(item.PKey, item.CCols, item.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *appStorage) readPartSort(ctx context.Context, part map[string][]byte, startCCols, finishCCols []byte) (sortKeys []string) {
	sortKeys = make([]string, 0)
	for col := range part {
		if ctx.Err() != nil {
			return nil
		}
		if len(startCCols) > 0 {
			if bytes.Compare(startCCols, []byte(col)) > 0 {
				continue
			}
		}
		if len(finishCCols) > 0 {
			if bytes.Compare([]byte(col), finishCCols) >= 0 {
				continue
			}
		}
		sortKeys = append(sortKeys, col)
	}
	sort.Strings(sortKeys)
	return sortKeys
}

func (s *appStorage) readPart(ctx context.Context, pKey []byte, startCCols, finishCCols []byte) (cCols, values [][]byte) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var (
		v  map[string][]byte
		ok bool
	)
	if v, ok = s.storage[string(pKey)]; !ok {
		return nil, nil // no such pKey
	}

	sortKeys := s.readPartSort(ctx, v, startCCols, finishCCols)
	if sortKeys == nil {
		return nil, nil
	}

	cCols = make([][]byte, 0)
	values = make([][]byte, 0)
	for _, col := range sortKeys {
		if ctx.Err() != nil {
			return nil, nil
		}
		cCols = append(cCols, copySlice([]byte(col)))
		values = append(values, copySlice(v[col]))
	}

	return cCols, values
}

func (s *appStorage) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb ReadCallback) (err error) {

	if (len(startCCols) > 0) && (len(finishCCols) > 0) && (bytes.Compare(startCCols, finishCCols) >= 0) {
		return nil // absurd range
	}

	if cCols, values := s.readPart(ctx, pKey, startCCols, finishCCols); cCols != nil {
		for i, cCol := range cCols {
			if ctx.Err() != nil {
				return nil
			}
			if err = cb(cCol, values[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *appStorage) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	p, ok := s.storage[string(pKey)]
	if !ok {
		return
	}
	viewRecord, ok := p[string(cCols)]
	if !ok {
		return
	}
	*data = append((*data)[0:0], copySlice(viewRecord)...)
	return
}

func (s *appStorage) GetBatch(pKey []byte, items []GetBatchItem) (err error) {
	for i := range items {
		items[i].Ok, err = s.Get(pKey, items[i].CCols, items[i].Data)
		if err != nil {
			return
		}
	}
	return
}

func copySlice(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
