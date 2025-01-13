/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
*/
package istoragecache

import (
	"testing"

	"github.com/voedger/voedger/pkg/coreutils"
	istructs "github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

/*
Before:
goos: linux
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istoragecache
cpu: 12th Gen Intel(R) Core(TM) i7-12700
BenchmarkAppStorage_Metrics
BenchmarkAppStorage_Metrics/GET
BenchmarkAppStorage_Metrics/GET-20         	 4889203	       239.9 ns/op	       8 B/op	       1 allocs/op
BenchmarkAppStorage_Metrics/GET-20         	11087750	       102.1 ns/op	       8 B/op	       1 allocs/op

After makeKey() optimization:
cpu: 12th Gen Intel(R) Core(TM) i7-12700
BenchmarkAppStorage_Get_Seq-20          	16371300			72.30 ns/op        0 B/op          0 allocs/op
*/
func BenchmarkAppStorage_Get_Seq(b *testing.B) {
	testData := []byte("atestdata")
	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = testData
		return true, nil
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "vvm", coreutils.NewITime())
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	pk := make([]byte, 40)
	cc := make([]byte, 30)
	var res []byte

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if !ok {
			panic("not ok")
		}
		if err != nil {
			panic(err)
		}

	}
}

func BenchmarkAppStorage_Get_Parallel(b *testing.B) {
	testData := []byte("atestdata")
	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = testData
		return true, nil
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "vvm", coreutils.NewITime())
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	pk := make([]byte, 40)
	cc := make([]byte, 30)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var res []byte
		for pb.Next() {
			ok, err1 := storage.Get(pk, cc, &res)
			if !ok {
				panic("not ok")
			}
			if err1 != nil {
				panic(err)
			}

		}
	})
}
