/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 * @author: Maxim Geraskin (refactoring)
 */

package bbolt

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	params := prepareTestData()
	defer cleanupTestData(params)

	// creating a StorageProvider
	factory := Provide(params, testingu.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	// get the required AppStorage for the app
	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	// write the application data to the database
	err = appStorage.Put([]byte("pKey"), []byte("cCols"), []byte("test data string"))
	require.NoError(err)

	// read the data from the database
	value := make([]byte, 0)
	ok, err := appStorage.Get([]byte("pKey"), []byte("cCols"), &value)
	require.True(ok)
	require.NoError(err)
	require.Equal([]byte("test data string"), value)
}

func TestTCK(t *testing.T) {
	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, testingu.MockTime)
	istorage.TechnologyCompatibilityKit(t, factory)
}

func Test_PutGet(t *testing.T) {
	require := require.New(t)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, testingu.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	appStorage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	err = appStorage.Put([]byte("persons"), []byte("NNV"), []byte("Nikitin Nikolay Valeryevich"))
	require.NoError(err)

	err = appStorage.Put([]byte("persons"), []byte("MDA"), []byte("Molchanovsky Dmitry Anatolyevich"))
	require.NoError(err)

	value := make([]byte, 0)

	ok, err := appStorage.Get([]byte("persons"), []byte("NNV"), &value)
	require.NoError(err)
	require.True(ok)
	require.Equal("Nikitin Nikolay Valeryevich", string(value))

	ok, err = appStorage.Get([]byte("persons"), []byte("MDA"), &value)
	require.NoError(err)
	require.True(ok)
	require.Equal("Molchanovsky Dmitry Anatolyevich", string(value))
}

func TestBackgroundCleaner(t *testing.T) {
	params := prepareTestData()
	defer cleanupTestData(params)

	r := require.New(t)
	iTime := testingu.NewMockTime()
	factory := Provide(params, iTime)
	storageProvider := istorageimpl.Provide(factory)

	firstTimerArmed := make(chan struct{})
	iTime.SetOnNextTimerArmed(func() { close(firstTimerArmed) })

	storage, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	r.NoError(err)
	<-firstTimerArmed

	// expires in 1 hour (50*60 = 3000s < 3600s)
	ok, err := storage.InsertIfNotExists([]byte("pKey"), []byte("cCols1"), []byte("value1"), 50*60)
	r.NoError(err)
	r.True(ok)
	// does NOT expire in 1 hour (61*60 = 3660s > 3600s)
	ok, err = storage.InsertIfNotExists([]byte("pKey"), []byte("cCols2"), []byte("value2"), 61*60)
	r.NoError(err)
	r.True(ok)
	// expires in 1 hour — nil clustering columns (exercises safeKey path in makeTTLKey/removeKey)
	ok, err = storage.InsertIfNotExists([]byte("pKey2"), nil, []byte("value3"), 50*60)
	r.NoError(err)
	r.True(ok)

	cleanerDone := make(chan struct{})
	iTime.SetOnNextNewTimerChan(func() { close(cleanerDone) })

	iTime.Sleep(time.Hour)
	<-cleanerDone

	value := make([]byte, 0)
	ok, err = storage.TTLGet([]byte("pKey"), []byte("cCols2"), &value)
	r.NoError(err)
	r.True(ok)

	// cleanerDone guarantees the cleanup transaction committed; verify physical deletion
	// wait for deletion
	impl := storage.(*appStorageType)
	r.Eventually(func() bool {
		var deleted bool
		viewErr := impl.db.View(func(tx *bolt.Tx) error {
			dataBucket := tx.Bucket([]byte(dataBucketName))
			if dataBucket == nil {
				// if data bucket is gone, key is gone
				deleted = true
				return nil
			}
			bucket := dataBucket.Bucket([]byte("pKey"))
			if bucket == nil {
				// if pkey bucket is gone, key is gone
				deleted = true
				return nil
			}
			toBeDeleted := bucket.Get(safeKey([]byte("cCols1")))
			deleted = len(toBeDeleted) == 0
			return nil
		})
		r.NoError(viewErr)
		return deleted
	}, time.Second, 50*time.Millisecond, "cCols1 not deleted from data bucket after TTL expiration")
	err = impl.db.View(func(tx *bolt.Tx) error {
		dataBucket := tx.Bucket([]byte(dataBucketName))
		r.NotNil(dataBucket)
		bucket := dataBucket.Bucket([]byte("pKey"))
		r.Nil(bucket.Get(safeKey([]byte("cCols1"))), "cCols1 not deleted from data bucket after TTL expiration")

		// pKey2 had only the nil-cCols entry; removeKey must have deleted the whole sub-bucket
		r.Nil(dataBucket.Bucket([]byte("pKey2")), "pKey2 bucket not deleted after nil-cCols TTL expiration")
		return nil
	})
	r.NoError(err)
}

func prepareTestData() (params ParamsType) {
	dbDir, err := os.MkdirTemp("", "bolt")
	if err != nil {
		panic(err)
	}
	params.DBDir = dbDir
	return
}

func cleanupTestData(params ParamsType) {
	if params.DBDir != "" {
		os.RemoveAll(params.DBDir)
		params.DBDir = ""
	}
}

func TestAppStorageFactory_StopGoroutines(t *testing.T) {
	require := require.New(t)

	params := prepareTestData()
	defer cleanupTestData(params)

	factory := Provide(params, testingu.MockTime)
	storageProvider := istorageimpl.Provide(factory)

	_, err := storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	storageProvider.Stop()

	implFactory := factory.(*appStorageFactory)
	require.Error(implFactory.ctx.Err())

	_, err = storageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.ErrorIs(err, istorageimpl.ErrStoppingState)
}
