/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package dbcertcache

import "golang.org/x/crypto/acme/autocert"

func ProvideDBCache(storage RouterAppStoragePtr) autocert.Cache {
	return &autoCertDBCache{
		appStorage: storage,
	}
}
