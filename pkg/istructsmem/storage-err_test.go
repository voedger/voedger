/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"context"

	istorage "github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istorageimpl"
	"github.com/untillpro/voedger/pkg/istructs"
)

// testStorageType is test storage. Trained to return the specified error
type (
	scheduleErrorType struct {
		err         error
		pKey, cCols []byte
	}

	damageFuncType     func(*[]byte)
	scheduleDamageType struct {
		dam damageFuncType
		scheduleErrorType
	}

	testStorageType struct {
		storage  istorage.IAppStorage
		get, put scheduleErrorType
		damage   scheduleDamageType
	}

	testStorageProvider struct {
		testStorage *testStorageType
	}
)

func (tsp *testStorageProvider) AppStorage(appName istructs.AppQName) (structs istorage.IAppStorage, err error) {
	return tsp.testStorage, nil
}

func newTestStorageProvider(ts *testStorageType) istorage.IAppStorageProvider {
	return &testStorageProvider{testStorage: ts}
}

func newTestStorage() *testStorageType {
	s := testStorageType{get: scheduleErrorType{}, put: scheduleErrorType{}}
	asf := istorage.ProvideMem()
	sp := istorageimpl.Provide(asf)
	var err error
	if s.storage, err = sp.AppStorage(istructs.AppQName_test1_app1); err != nil {
		panic(err)
	}

	return &s
}

// occurs returns that primary key and clustering columns matches the shedduled error
func (e *scheduleErrorType) match(pKey, cCols []byte) bool {
	return ((len(e.pKey) == 0) || bytes.Equal(e.pKey, pKey)) &&
		((len(e.cCols) == 0) || bytes.Equal(e.cCols, cCols))
}

func (s *testStorageType) reset() {
	s.get = scheduleErrorType{}
	s.put = scheduleErrorType{}
	s.damage = scheduleDamageType{}
}

func (s *testStorageType) sheduleGetError(err error, pKey, cCols []byte) {
	s.get.err = err
	s.get.pKey = make([]byte, len(pKey))
	copy(s.get.pKey, pKey)
	s.get.cCols = make([]byte, len(cCols))
	copy(s.get.cCols, cCols)
}

func (s *testStorageType) sheduleGetDamage(dam damageFuncType, pKey, cCols []byte) {
	s.damage.dam = dam
	s.damage.pKey = make([]byte, len(pKey))
	copy(s.damage.pKey, pKey)
	s.damage.cCols = make([]byte, len(cCols))
	copy(s.damage.cCols, cCols)
}

func (s *testStorageType) shedulePutError(err error, pKey, cCols []byte) {
	s.put.err = err
	s.put.pKey = make([]byte, len(pKey))
	copy(s.put.pKey, pKey)
	s.put.cCols = make([]byte, len(cCols))
	copy(s.put.cCols, cCols)
}

func (s *testStorageType) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	if s.get.err != nil {
		if s.get.match(pKey, cCols) {
			err = s.get.err
			s.get.err = nil
			return false, err
		}
	}

	ok, err = s.storage.Get(pKey, cCols, data)

	if ok && (s.damage.dam != nil) {
		if s.damage.match(pKey, cCols) {
			s.damage.dam(data)
			s.damage.dam = nil
			return ok, err
		}
	}

	return ok, err
}

func (s *testStorageType) GetBatch(pKey []byte, items []istorage.GetBatchItem) (err error) {
	if s.get.err != nil {
		for _, item := range items {
			if s.get.match(pKey, item.CCols) {
				err = s.get.err
				s.get.err = nil
				return err
			}
		}
	}

	err = s.storage.GetBatch(pKey, items)

	if s.damage.dam != nil {
		for i := 0; i < len(items); i++ {
			if s.damage.match(pKey, items[i].CCols) {
				if items[i].Ok {
					s.damage.dam(items[i].Data)
					s.damage.dam = nil
				}
			}
		}
	}

	return err
}

func (s *testStorageType) Put(pKey []byte, cCols []byte, value []byte) (err error) {
	if s.put.err != nil {
		if s.put.match(pKey, cCols) {
			err = s.put.err
			s.put.err = nil
			return err
		}
	}
	return s.storage.Put(pKey, cCols, value)
}

func (s *testStorageType) PutBatch(items []istorage.BatchItem) (err error) {
	for _, p := range items {
		if err = s.Put(p.PKey, p.CCols, p.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *testStorageType) Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb istorage.ReadCallback) (err error) {
	cbWrap := func(cCols []byte, data []byte) (err error) {
		if s.get.err != nil {
			if s.get.match(pKey, cCols) {
				err = s.get.err
				s.get.err = nil
				return err
			}
		}

		if s.damage.dam != nil {
			if s.damage.match(pKey, cCols) {
				s.damage.dam(&data)
				s.damage.dam = nil
			}
		}

		return cb(cCols, data)
	}

	return s.storage.Read(ctx, pKey, startCCols, finishCCols, cbWrap)
}
