/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

// nolint
package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func (asi *implIAppStorageInitializer) Init(appQName istructs.AppQName) (err error) {
	asi.lock.Lock()
	defer asi.lock.Unlock()
	if _, ok := asi.cache[appQName]; ok {
		return ErrStorageInitedAlready
	}

	if asi.metaStorage == nil {
		if asi.metaStorage, err = asi.getMetaStorage(); err != nil {
			return err
		}
	}

	exists, appStorageDesc, err := readAppStorageDesc(appQName, asi.metaStorage)
	if err != nil {
		return err
	}
	if !exists {
		if appStorageDesc, err = getNewAppStorageDesc(appQName, asi.metaStorage); err != nil {
			return err
		}
	}

	if len(appStorageDesc.Error) == 0 && appStorageDesc.Status == istorage.AppStorageStatus_Pending {
		if err := asi.asf.Init(asi.clarifyKeyspaceName(appStorageDesc.SafeName)); err != nil {
			appStorageDesc.Error = err.Error()
		} else {
			appStorageDesc.Status = istorage.AppStorageStatus_Done
		}
		// possible: new SafeAppName written , but appDesc write is failed. No problem in this case because we'll just have an orphaned record
		if err = storeAppDesc(appQName, appStorageDesc, asi.metaStorage); err != nil {
			return err
		}
	}
	if len(appStorageDesc.Error) > 0 {
		return fmt.Errorf("%s: %w: %s", appStorageDesc.SafeName.String(), ErrStorageInitError, appStorageDesc.Error)
	}
	storage, err := asi.asf.AppStorage(asi.clarifyKeyspaceName(appStorageDesc.SafeName))
	if err == nil {
		asi.cache[appQName] = storage
	}
	return err

}

func (asi *implIAppStorageInitializer) AppStorage(appQName istructs.AppQName) (storage istorage.IAppStorage, err error) {
	asi.lock.Lock()
	defer asi.lock.Unlock()
	if storage, ok := asi.cache[appQName]; ok {
		return storage, nil
	}
	return nil, fmt.Errorf("%s: %w", appQName, ErrStorageNotInited)
}

func (asi *implIAppStorageInitializer) getMetaStorage() (istorage.IAppStorage, error) {
	if err := asi.asf.Init(asi.clarifyKeyspaceName(istorage.SysMetaSafeName)); err != nil && err != istorage.ErrStorageAlreadyExists {
		return nil, err
	}
	return asi.asf.AppStorage(asi.clarifyKeyspaceName(istorage.SysMetaSafeName))
}

func (asi *implIAppStorageInitializer) clarifyKeyspaceName(sn istorage.SafeAppName) istorage.SafeAppName {
	if coreutils.IsTest() {
		// unique safe keyspace name is generated at istorage.NewSafeAppName()
		// uuid suffix is need in tests only avoiding the case:
		// - go test ./... in github using Scylla
		// - integration tests for different packages are run in simultaneously in separate processes
		// - 2 processes using the same shared VIT config -> 2 VITs are initialized on the same keyspaces names -> conflict when e.g. creating the same logins
		// see also getNewAppStorageDesc() below
		newName := sn.String() + asi.suffix
		newName = strings.ReplaceAll(newName, "-", "")
		if len(newName) > istorage.MaxSafeNameLength {
			newName = newName[:istorage.MaxSafeNameLength]
		}
		sn = istorage.NewTestSafeName(newName)
	}
	return sn
}

func storeAppDesc(appQName istructs.AppQName, appDesc istorage.AppStorageDesc, metaStorage istorage.IAppStorage) error {
	pkBytes := []byte(appQName.String())
	cColsBytes := cCols_AppStorageDesc
	appDescJSON, err := json.Marshal(&appDesc)
	if err != nil {
		// notest
		return err
	}
	return metaStorage.Put(pkBytes, cColsBytes, appDescJSON)
}

func getNewAppStorageDesc(appQName istructs.AppQName, metaStorage istorage.IAppStorage) (res istorage.AppStorageDesc, err error) {
	san, err := istorage.NewSafeAppName(appQName, func(name string) (bool, error) {
		pkBytes := []byte(name)
		exists, err := metaStorage.Get(pkBytes, cCols_SafeAppName, &value_SafeAppName)
		if err != nil {
			return false, err
		}
		return !exists, nil
	})
	if err != nil {
		return res, err
	}
	// store new SafeAppName
	pkBytes := []byte(san.String())
	if err := metaStorage.Put(pkBytes, cCols_SafeAppName, value_SafeAppName); err != nil {
		return res, err
	}
	return istorage.AppStorageDesc{
		SafeName: san,
		Status:   istorage.AppStorageStatus_Pending,
	}, nil
}

func readAppStorageDesc(appQName istructs.AppQName, metaStorage istorage.IAppStorage) (ok bool, appStorageDesc istorage.AppStorageDesc, err error) {
	pkBytes := []byte(appQName.String())
	appDescJSON := []byte{}
	if ok, err = metaStorage.Get(pkBytes, cCols_AppStorageDesc, &appDescJSON); err != nil {
		return
	}
	if ok {
		err = json.Unmarshal(appDescJSON, &appStorageDesc)
	}
	return
}
