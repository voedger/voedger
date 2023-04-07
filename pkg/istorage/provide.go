/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istorage

func ProvideMem() IAppStorageFactory {
	return &appStorageFactory{storages: map[string]map[string]map[string][]byte{}}
}
