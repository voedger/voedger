/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
*/
package istoragecache

import (
	"testing"

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

After:
TBD
*/
func BenchmarkAppStorage_Metrics(b *testing.B) {
	testData := []byte("atestdata")
	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = testData
		return true, nil
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testConf, tsp, imetrics.Provide(), "vvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	pk := []byte("pk")
	cc := []byte("cc")
	var res []byte

	b.Run("GET", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ok, err := storage.Get(pk, cc, &res)
			if !ok {
				panic("not ok")
			}
			if err != nil {
				panic(err)
			}

		}
	})
}
