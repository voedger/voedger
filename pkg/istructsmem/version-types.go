/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
)

type (
	// versionKeyType type for version key, see ver××× consts
	versionKeyType uint16

	// versionValueType type for version values, see ver××× consts
	versionValueType uint16

	// verionsCacheType type to cache versions of system views
	verionsCacheType struct {
		app  *AppConfigType
		vers map[versionKeyType]versionValueType
	}
)

// newVerionsCache constructs new versions cache
func newVerionsCache(app *AppConfigType) verionsCacheType {
	ver := verionsCacheType{app: app, vers: make(map[versionKeyType]versionValueType)}
	return ver
}

// prepare prepare cache for all versions of system views
func (vers *verionsCacheType) prepare() (err error) {
	pKey := toBytes(uint16(QNameIDSysVesions))
	return vers.app.storage.Read(context.Background(), pKey, nil, nil,
		func(cCols, value []byte) (_ error) {
			var (
				key versionKeyType
				val versionValueType
			)
			key = versionKeyType(binary.BigEndian.Uint16(cCols))
			val = versionValueType(binary.BigEndian.Uint16(value))
			vers.vers[key] = val
			return nil
		})
}

// getVersion returns version value for version key
func (vers *verionsCacheType) getVersion(key versionKeyType) versionValueType {
	return vers.vers[key]
}

// putVersion stores version value for version key into application storage
func (vers *verionsCacheType) putVersion(key versionKeyType, value versionValueType) (err error) {
	vers.vers[key] = value

	return vers.app.storage.Put(
		toBytes(uint16(QNameIDSysVesions)),
		toBytes(uint16(key)),
		toBytes(uint16(value)),
	)
}
