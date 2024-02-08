/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

import (
	"context"
	"encoding/binary"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

func newVersions() *Versions {
	return &Versions{vers: make(map[VersionKey]VersionValue)}
}

// Returns version value for version key
func (vers *Versions) Get(key VersionKey) VersionValue {
	return vers.vers[key]
}

// Stores version value for version key into application storage
func (vers *Versions) Put(key VersionKey, value VersionValue) (err error) {
	vers.vers[key] = value

	return vers.storage.Put(
		utils.ToBytes(consts.SysView_Versions),
		utils.ToBytes(uint16(key)),
		utils.ToBytes(uint16(value)),
	)
}

// Prepares cache for all versions of system views
func (vers *Versions) Prepare(storage istorage.IAppStorage) (err error) {
	vers.storage = storage
	pKey := utils.ToBytes(consts.SysView_Versions)
	return vers.storage.Read(context.Background(), pKey, nil, nil,
		func(cCols, value []byte) (_ error) {
			key := VersionKey(binary.BigEndian.Uint16(cCols))
			val := VersionValue(binary.BigEndian.Uint16(value))
			vers.vers[key] = val
			return nil
		})
}
