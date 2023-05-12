/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

// uniques IDs system view.
type uniques struct {
	ids     map[string]appdef.UniqueID
	lastID  appdef.UniqueID
	changes uint
	qnames  *qnames.QNames
}

func newUniques() *uniques {
	return &uniques{
		ids:    make(map[string]appdef.UniqueID),
		lastID: appdef.FirstUniqueID,
	}
}

// Loads all uniques IDs from storage, add all uniques from application definitions and store if some changes.
// Must be called at application starts
func (un *uniques) prepare(storage istorage.IAppStorage, versions *vers.Versions, qnames *qnames.QNames, appDef appdef.IAppDef) (err error) {
	if err = un.load(storage, versions); err != nil {
		return err
	}

	un.qnames = qnames

	if appDef != nil {
		if err = un.collectAll(appDef); err != nil {
			return err
		}
	}

	if un.changes > 0 {
		if err := un.store(storage, versions); err != nil {
			return err
		}
	}

	return nil
}

// Returns key for unique.
//
// Keys structure:
//   - definition QNameID in 4-digit hexadecimal form, e.g. "0x07b5"
//   - field names pipe-separated concatenation, e.g. "|name|surname|"
//
// e.g. "0x07b5|name|surname|"
func (un *uniques) key(u appdef.IUnique) (string, error) {
	id, err := un.qnames.ID(u.Def().QName())
	if err != nil {
		return "", err
	}

	const (
		pipe  = "|"
		idFmt = "%#.4x" + pipe // Four digits, QNameID based on uint16
	)

	s := fmt.Sprintf(idFmt, id)
	for _, f := range u.Fields() {
		s += f.Name()
		s += pipe
	}
	return s, nil
}

// loads all stored unique IDs from storage
func (un *uniques) load(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	ver := versions.Get(vers.SysUniquesVersion)
	switch ver {
	case vers.UnknownVersion: // no system uniques view exists in storage
		return nil
	case ver01:
		return un.load01(storage)
	}

	return fmt.Errorf("unable load uniques IDs from system view version %v: %w", ver, vers.ErrorInvalidVersion)
}

// Loads uniques IDs from storage using ver01 codec
func (un *uniques) load01(storage istorage.IAppStorage) error {

	readUnique := func(cCols, value []byte) error {
		k := string(cCols)
		id := appdef.UniqueID(binary.BigEndian.Uint32(value))
		un.ids[k] = id

		if un.lastID < id {
			un.lastID = id
		}

		return nil
	}

	pKey := utils.ToBytes(consts.SysView_UniquesIDs, ver01)
	return storage.Read(context.Background(), pKey, nil, nil, readUnique)
}

// Collect unique
func (un *uniques) collect(u appdef.IUnique) (err error) {
	k, err := un.key(u)
	if err != nil {
		return err
	}

	id, exists := un.ids[k]
	if !exists {
		un.lastID++
		id = un.lastID
		un.ids[k] = id
		un.changes++
	}

	u.(interface{ SetID(appdef.UniqueID) }).SetID(id)

	return nil
}

// Collect all application uniques
func (un *uniques) collectAll(appDef appdef.IAppDef) (err error) {
	appDef.Defs(
		func(d appdef.IDef) {
			d.Uniques(func(u appdef.IUnique) {
				err = errors.Join(err, un.collect(u))
			})
		})

	return err
}

// stores uniques IDs using latestVersion codec
func (un *uniques) store(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	pKey := utils.ToBytes(consts.SysView_UniquesIDs, latestVersion)

	batch := make([]istorage.BatchItem, 0)
	for k, id := range un.ids {
		if id >= appdef.FirstUniqueID {
			item := istorage.BatchItem{
				PKey:  pKey,
				CCols: []byte(k),
				Value: utils.ToBytes(id),
			}
			batch = append(batch, item)
		}
	}

	if err = storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application unique IDs to storage: %w", err)
	}

	if ver := versions.Get(vers.SysUniquesVersion); ver != latestVersion {
		if err = versions.Put(vers.SysUniquesVersion, latestVersion); err != nil {
			return fmt.Errorf("error store uniques system view version: %w", err)
		}
	}

	un.changes = 0
	return nil
}
