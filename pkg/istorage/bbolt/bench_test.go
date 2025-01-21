/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package bbolt

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func Benchmark_Put_One_SameBucket_ST(b *testing.B) {
	require := require.New(b)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, coreutils.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	var cCols = make([]byte, 8)

	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(cCols, rand.Uint64())
		err = appStorage.Put([]byte("persons"), cCols, []byte("Nikitin Nikolay Valeryevich"))
		if err != nil {
			panic(err)
		}
	}
}

func Benchmark_Put_50_DifferentBuckets_ST(b *testing.B) {

	const NumOfBatchItems = 50

	require := require.New(b)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, coreutils.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	var pKey = make([]byte, 8)
	var cCols = make([]byte, 8)
	var batchItems = make([]istorage.BatchItem, NumOfBatchItems)

	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(pKey, rand.Uint64())

		for j := 0; j < NumOfBatchItems; j++ {
			binary.BigEndian.PutUint64(cCols, rand.Uint64())
			batchItems[j] = istorage.BatchItem{PKey: pKey, CCols: cCols, Value: []byte("Nikitin Nikolay Valeryevich")}
		}
		err = appStorage.PutBatch(batchItems)
		if err != nil {
			panic(err)
		}
	}
}

func Benchmark_Put_One_DifferentBuckets_ST(b *testing.B) {
	require := require.New(b)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, coreutils.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	var pKey = make([]byte, 8)
	var cCols = make([]byte, 8)

	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(pKey, rand.Uint64())
		binary.BigEndian.PutUint64(cCols, rand.Uint64())
		err = appStorage.Put(pKey, cCols, []byte("Nikitin Nikolay Valeryevich"))
		if err != nil {
			panic(err)
		}
	}
}

func Benchmark_Put_One_SameBucket_Parallel(b *testing.B) {

	require := require.New(b)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, coreutils.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := appStorage.Put([]byte("persons"), []byte("NNV"), []byte("Nikitin Nikolay Valeryevich"))
			if err != nil {
				panic(err)
			}
		}
	})
}
