/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package dbcertcache

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"reflect"

	"golang.org/x/crypto/acme/autocert"
)

// Get return certificate for the specified key(domain).
// If there's no such key, Get returns ErrCacheMiss.
func (ac *autoCertDBCache) Get(ctx context.Context, key string) (data []byte, err error) {
	domainKey, err := createKey(CertPrefixInRouterStorage, key)
	if err != nil {
		return nil, err
	}
	ok, err := (*(ac.appStorage)).Get(domainKey.Bytes(), nil, &data)
	if err != nil {
		return nil, err
	}
	if !ok || len(data) == 0 {
		return nil, autocert.ErrCacheMiss
	}
	return data, err
}

// Put stores the data in the cache under the specified key.
func (ac *autoCertDBCache) Put(ctx context.Context, key string, data []byte) (err error) {
	domainKey, err := createKey(CertPrefixInRouterStorage, key)
	if err != nil {
		return err
	}
	return (*(ac.appStorage)).Put(domainKey.Bytes(), nil, data)
}

// Delete IAppStorage does not have Delete method, therefore set value to nil and in Get method check value for length
func (ac *autoCertDBCache) Delete(ctx context.Context, key string) (err error) {
	domainKey, err := createKey(CertPrefixInRouterStorage, key)
	if err != nil {
		return err
	}
	return (*(ac.appStorage)).Put(domainKey.Bytes(), nil, nil)
}

func createKey(columns ...interface{}) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	for _, col := range columns {
		switch v := col.(type) {
		case uint16:
			if err = binary.Write(buf, binary.BigEndian, v); err != nil {
				// error impossible
				// notest
				return buf, nil
			}
		case string:
			if err = binary.Write(buf, binary.LittleEndian, []byte(v)); err != nil {
				// error impossible
				// notest
				return buf, err
			}
		default:
			return nil, fmt.Errorf("unsupported data type %s: %w", reflect.ValueOf(col).Type(), ErrPKeyCreateError)
		}
	}
	return buf, nil
}
