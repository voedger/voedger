/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package provider

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
)

// keyspaceIsolationSuffix is used in tests only
// need to make different keyspaces when few integration tests run on the same non-memory storage. E.g. on `go test ./...` in github
// otherwise tests from different packages will conflict
// normally should be a unique random string per VVM
// see https://dev.untill.com/projects/#!638565
func Provide(asf istorage.IAppStorageFactory, keyspaceIsolationSuffix ...string) istorage.IAppStorageProvider {
	res := &implIAppStorageProvider{
		asf:   asf,
		cache: map[appdef.AppQName]istorage.IAppStorage{},
	}
	if len(keyspaceIsolationSuffix) > 0 && len(keyspaceIsolationSuffix[0]) > 0 {
		res.keyspaceIsolationSuffix = keyspaceIsolationSuffix[0]
	}
	var err error
	res.sysMetaAppSafeName, err = istorage.NewSafeAppName(appdef.NewAppQName(appdef.SysPackage, "meta"), func(name string) (bool, error) { return true, nil })
	if err != nil {
		// notest
		panic(err)
	}
	res.sysMetaAppSafeName.ApplyKeyspaceIsolationSuffix(res.keyspaceIsolationSuffix)
	return res
}
