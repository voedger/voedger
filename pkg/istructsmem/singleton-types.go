/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	istorage "github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/utils"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/vers"
	"github.com/untillpro/voedger/pkg/schemas"
)

// singletonsCacheType is cache for CDoc singleton IDs
type singletonsCacheType struct {
	cfg     *AppConfigType
	qNames  map[istructs.QName]istructs.RecordID
	ids     map[istructs.RecordID]istructs.QName
	lastID  istructs.RecordID
	changes uint32
}

func newSingletonsCache(cfg *AppConfigType) singletonsCacheType {
	return singletonsCacheType{
		cfg:    cfg,
		qNames: make(map[istructs.QName]istructs.RecordID),
		ids:    make(map[istructs.RecordID]istructs.QName),
		lastID: istructs.FirstSingletonID - 1,
	}
}

// clear clears singletons cache
func (stons *singletonsCacheType) clear() {
	stons.qNames = make(map[istructs.QName]istructs.RecordID)
	stons.ids = make(map[istructs.RecordID]istructs.QName)
	stons.lastID = istructs.FirstSingletonID - 1
	stons.changes = 0
}

// collectAllSingletons collect all application singlton IDs
func (stons *singletonsCacheType) collectAllSingletons() (err error) {
	stons.cfg.Schemas.EnumSchemas(
		func(schema *schemas.Schema) {
			if schema.Singleton() {
				err = errors.Join(err,
					stons.collectSingleton(schema))
			}
		})

	return err
}

// collectSingleton checks is application schema singleton in cache. If not then adds it with new ID
func (stons *singletonsCacheType) collectSingleton(schema *schemas.Schema) (err error) {

	name := schema.QName()

	if _, ok := stons.qNames[name]; ok {
		return nil // already known schema
	}

	for id := stons.lastID + 1; id < istructs.MaxSingletonID; id++ {
		if _, ok := stons.ids[id]; !ok {
			stons.qNames[name] = id
			stons.ids[id] = name
			stons.lastID = id
			stons.changes++
			return nil
		}
	}

	return ErrSingletonIDsExceeds
}

// idToQName returns QName for specified singleton ID
func (stons *singletonsCacheType) idToQName(id istructs.RecordID) (istructs.QName, error) {
	name, ok := stons.ids[id]
	if ok {
		return name, nil
	}

	return istructs.NullQName, fmt.Errorf("unknown singleton ID «%v»: %w", id, ErrIDNotFound)
}

// load loads all stored singleton IDs from storage
func (stons *singletonsCacheType) load() (err error) {
	stons.clear()

	ver := stons.cfg.versions.GetVersion(vers.SysSingletonsVersion)
	switch ver {
	case vers.UnknownVersion: // no sys.QName storage exists
		return nil
	case verSysSingletonsLastest:
		return stons.load01()
	}

	return fmt.Errorf("unable load singleton IDs from «sys.Singletons» system view version %v: %w", ver, ErrorInvalidVersion)
}

// load01 loads all stored singketon IDs from storage using verSysSingletons01 codec
func (stons *singletonsCacheType) load01() error {

	readSingleton := func(cCols, value []byte) error {
		qName, err := istructs.ParseQName(string(cCols))
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

	pKey := utils.ToBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletons01))
	return stons.cfg.storage.Read(context.Background(), pKey, nil, nil, readSingleton)
}

// prepare loads all singleton IDs from storage, add all known application singletons and store cache if some changes. Must be called at application starts
func (stons *singletonsCacheType) prepare() (err error) {
	if err = stons.load(); err != nil {
		return err
	}

	if err = stons.collectAllSingletons(); err != nil {
		return err
	}

	if stons.changes > 0 {
		if err := stons.store(); err != nil {
			return err
		}
	}

	return nil
}

// qNameToID returns ID for specified CDOC document
func (stons *singletonsCacheType) qNameToID(qName istructs.QName) (istructs.RecordID, error) {
	if id, ok := stons.qNames[qName]; ok {
		return id, nil
	}
	return istructs.NullRecordID, fmt.Errorf("unable to find singleton ID for schema «%v»: %w", qName, ErrNameNotFound)
}

// store stores all known singleton IDs to storage using verSysSingletonsLastest codec
func (stons *singletonsCacheType) store() (err error) {
	pKey := utils.ToBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletonsLastest))

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

	if err = stons.cfg.storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application singleton IDs to storage: %w", err)
	}

	if ver := stons.cfg.versions.GetVersion(vers.SysSingletonsVersion); ver != verSysSingletonsLastest {
		if err = stons.cfg.versions.PutVersion(vers.SysSingletonsVersion, verSysSingletonsLastest); err != nil {
			return fmt.Errorf("error store «sys.Singletons» system view version: %w", err)
		}
	}

	stons.changes = 0
	return nil
}
