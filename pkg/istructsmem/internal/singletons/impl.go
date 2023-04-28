/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func newSingletons() *Singletons {
	return &Singletons{
		qNames: make(map[appdef.QName]istructs.RecordID),
		ids:    make(map[istructs.RecordID]appdef.QName),
		lastID: istructs.FirstSingletonID - 1,
	}
}

// Returns QName for CDoc singleton with specified ID
func (stons *Singletons) GetQName(id istructs.RecordID) (appdef.QName, error) {
	name, ok := stons.ids[id]
	if ok {
		return name, nil
	}

	return appdef.NullQName, fmt.Errorf("unknown singleton ID «%v»: %w", id, ErrIDNotFound)
}

// Returns ID for CDoc singleton with specified QName
func (stons *Singletons) GetID(qName appdef.QName) (istructs.RecordID, error) {
	if id, ok := stons.qNames[qName]; ok {
		return id, nil
	}
	return istructs.NullRecordID, fmt.Errorf("unable to find singleton ID for definition «%v»: %w", qName, ErrNameNotFound)
}

// Loads all singletons IDs from storage, add all known application singletons and store cache if some changes.
// Must be called at application starts
func (stons *Singletons) Prepare(storage istorage.IAppStorage, versions *vers.Versions, appDef appdef.IAppDef) (err error) {
	if err = stons.load(storage, versions); err != nil {
		return err
	}

	if appDef != nil {
		if err = stons.collectAllSingletons(appDef); err != nil {
			return err
		}
	}

	if stons.changes > 0 {
		if err := stons.store(storage, versions); err != nil {
			return err
		}
	}

	return nil
}

// loads all stored singleton IDs from storage
func (stons *Singletons) load(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	ver := versions.Get(vers.SysSingletonsVersion)
	switch ver {
	case vers.UnknownVersion: // no sys.QName storage exists
		return nil
	case ver01:
		return stons.load01(storage)
	}

	return fmt.Errorf("unable load singleton IDs from «sys.Singletons» system view version %v: %w", ver, vers.ErrorInvalidVersion)
}

// Loads singletons IDs from storage using ver01 codec
func (stons *Singletons) load01(storage istorage.IAppStorage) error {

	readSingleton := func(cCols, value []byte) error {
		qName, err := appdef.ParseQName(string(cCols))
		if err != nil {
			return err
		}
		id := istructs.RecordID(binary.BigEndian.Uint64(value))

		stons.qNames[qName] = id
		stons.ids[id] = qName

		if stons.lastID < id {
			stons.lastID = id
		}

		return nil
	}

	pKey := utils.ToBytes(consts.SysView_SingletonIDs, ver01)
	return storage.Read(context.Background(), pKey, nil, nil, readSingleton)
}

// Collect all application singlton IDs
func (stons *Singletons) collectAllSingletons(appDef appdef.IAppDef) (err error) {
	appDef.Defs(
		func(d appdef.IDef) {
			if d.Singleton() {
				err = errors.Join(err,
					stons.collectSingleton(d.QName()))
			}
		})

	return err
}

// collectSingleton checks is application definition singleton in cache. If not then adds it with new ID
func (stons *Singletons) collectSingleton(qname appdef.QName) error {

	if _, ok := stons.qNames[qname]; ok {
		return nil // already known singleton
	}

	for id := stons.lastID + 1; id < istructs.MaxSingletonID; id++ {
		if _, ok := stons.ids[id]; !ok {
			stons.qNames[qname] = id
			stons.ids[id] = qname
			stons.lastID = id
			stons.changes++
			return nil
		}
	}

	return ErrSingletonIDsExceeds
}

// stores singletons IDs using lastestVersion codec
func (stons *Singletons) store(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	pKey := utils.ToBytes(consts.SysView_SingletonIDs, lastestVersion)

	batch := make([]istorage.BatchItem, 0)
	for qName, id := range stons.qNames {
		if id >= istructs.FirstSingletonID {
			item := istorage.BatchItem{
				PKey:  pKey,
				CCols: []byte(qName.String()),
				Value: utils.ToBytes(uint64(id)),
			}
			batch = append(batch, item)
		}
	}

	if err = storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application singleton IDs to storage: %w", err)
	}

	if ver := versions.Get(vers.SysSingletonsVersion); ver != lastestVersion {
		if err = versions.Put(vers.SysSingletonsVersion, lastestVersion); err != nil {
			return fmt.Errorf("error store «sys.Singletons» system view version: %w", err)
		}
	}

	stons.changes = 0
	return nil
}
