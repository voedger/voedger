/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package dbcertcache

import "github.com/voedger/voedger/pkg/istorage"

type RouterAppStoragePtr *istorage.IAppStorage

type autoCertDBCache struct {
	appStorage *istorage.IAppStorage
}
