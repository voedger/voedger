/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package dbcertcache

import "golang.org/x/crypto/acme/autocert"

func ProvideDbCache(storage RouterAppStorage) autocert.Cache {
	return &autoCertDbCache{
		appStorage: storage,
	}
}
