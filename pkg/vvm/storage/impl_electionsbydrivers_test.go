/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/amazondb"
	"github.com/voedger/voedger/pkg/istorage/bbolt"
	"github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestTTLStorageMem(t *testing.T) {
	storageFactory := mem.Provide(coreutils.MockTime)
	TTLStorageTestSuite(t, storageFactory)
}

func TestTTStorageCas(t *testing.T) {
	if !coreutils.IsCassandraStorage() {
		t.Skip()
	}
	storagePactory, err := cas.Provide(cas.DefaultCasParams)
	require.NoError(t, err)
	TTLStorageTestSuite(t, storagePactory)
}

func TestTTLStorageBbolt(t *testing.T) {
	storagePactory := bbolt.Provide(bbolt.ParamsType{
		DBDir: os.TempDir(),
	}, coreutils.MockTime)
	TTLStorageTestSuite(t, storagePactory)
}

func TestTTLStorageDynamoDB(t *testing.T) {
	if !coreutils.IsDynamoDBStorage() {
		t.Skip()
	}
	storagePactory := amazondb.Provide(amazondb.DefaultDynamoDBParams, coreutils.MockTime)
	TTLStorageTestSuite(t, storagePactory)
}

func TTLStorageTestSuite(t *testing.T, appStorageFactory istorage.IAppStorageFactory) {
	appStroageProvider := provider.Provide(appStorageFactory)
	sysVVMAppStorage, err := appStroageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(t, err)
	electionsTTLStorage := NewElectionsTTLStorage(sysVVMAppStorage)
	counter := 0
	ielections.ElectionsTestSuite(t, electionsTTLStorage, ielections.TestDataGen[TTLStorageImplKey, string]{
		NextKey: func() TTLStorageImplKey {
			counter++
			return TTLStorageImplKey(counter)
		},
		NextVal: func() string {
			counter++
			return "testVal" + strconv.Itoa(counter)
		},
	})
}
