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

// Returns ID for singleton with specified QName
func (st *Singletons) ID(qName appdef.QName) (istructs.RecordID, error) {
	if id, ok := st.qNames[qName]; ok {
		return id, nil
	}
	return istructs.NullRecordID, fmt.Errorf("unable to find singleton ID for type «%v»: %w", qName, ErrNameNotFound)
}

// Loads all singletons IDs from storage, add all known application singletons and store if some changes.
// Must be called at application starts
func (st *Singletons) Prepare(storage istorage.IAppStorage, versions *vers.Versions, appDef appdef.IAppDef) (err error) {
	if err = st.load(storage, versions); err != nil {
		return err
	}

	if appDef != nil {
		if err = st.collectAllSingletons(appDef); err != nil {
			return err
		}
	}

	if st.changes > 0 {
		if err := st.store(storage, versions); err != nil {
			return err
		}
	}

	return nil
}

// loads all stored singleton IDs from storage
func (st *Singletons) load(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	ver := versions.Get(vers.SysSingletonsVersion)
	switch ver {
	case vers.UnknownVersion: // no system singletons view exists in storage
		return nil
	case ver01:
		return st.load01(storage)
	}

	return fmt.Errorf("unable load singleton IDs from system view version %v: %w", ver, vers.ErrorInvalidVersion)
}

// Loads singletons IDs from storage using ver01 codec
func (st *Singletons) load01(storage istorage.IAppStorage) error {

	readSingleton := func(cCols, value []byte) error {
		qName, err := appdef.ParseQName(string(cCols))
		if err != nil {
			return err
		}
		id := istructs.RecordID(binary.BigEndian.Uint64(value))

		st.qNames[qName] = id
		st.ids[id] = qName

		if st.lastID < id {
			st.lastID = id
		}

		return nil
	}

	pKey := utils.ToBytes(consts.SysView_SingletonIDs, ver01)
	return storage.Read(context.Background(), pKey, nil, nil, readSingleton)
}

// Collect all application singleton IDs
func (st *Singletons) collectAllSingletons(appDef appdef.IAppDef) (err error) {
	for s := range appdef.Singletons(appDef.Types) {
		if s.Singleton() {
			err = errors.Join(err,
				st.collectSingleton(s.QName()))
		}
	}

	return err
}

// collectSingleton checks is singleton in cache. If not then adds it with new ID
func (st *Singletons) collectSingleton(qname appdef.QName) error {

	if _, ok := st.qNames[qname]; ok {
		return nil // already known singleton
	}

	for id := st.lastID + 1; id < istructs.MaxSingletonID; id++ {
		if _, ok := st.ids[id]; !ok {
			st.qNames[qname] = id
			st.ids[id] = qname
			st.lastID = id
			st.changes++
			return nil
		}
	}

	return ErrSingletonIDsExceeds
}

// stores singletons IDs using latestVersion codec
func (st *Singletons) store(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	pKey := utils.ToBytes(consts.SysView_SingletonIDs, latestVersion)

	batch := make([]istorage.BatchItem, 0)
	for qName, id := range st.qNames {
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

	if ver := versions.Get(vers.SysSingletonsVersion); ver != latestVersion {
		if err = versions.Put(vers.SysSingletonsVersion, latestVersion); err != nil {
			return fmt.Errorf("error store singletons system view version: %w", err)
		}
	}

	st.changes = 0
	return nil
}
