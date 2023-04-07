/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
	"fmt"

	istorage "github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
)

// singletonsCacheType is cache for CDoc singleton IDs
type singletonsCacheType struct {
	app     *AppConfigType
	qNames  map[istructs.QName]istructs.RecordID
	ids     map[istructs.RecordID]istructs.QName
	lastID  istructs.RecordID
	changes uint32
}

func newSingletonsCache(app *AppConfigType) singletonsCacheType {
	return singletonsCacheType{
		app:    app,
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
	for _, sch := range stons.app.Schemas.schemas {
		if sch.singleton.enabled {
			if err = stons.collectSingleton(sch); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectSingleton checks is application schema singleton in cache. If not then adds it with new ID
func (stons *singletonsCacheType) collectSingleton(schema *SchemaType) (err error) {

	if id, ok := stons.qNames[schema.name]; ok {
		schema.singleton.id = id
		return nil // already known schema
	}

	for id := stons.lastID + 1; id < istructs.MaxSingletonID; id++ {
		if _, ok := stons.ids[id]; !ok {
			stons.qNames[schema.name] = id
			stons.ids[id] = schema.name
			stons.lastID = id
			stons.changes++
			schema.singleton.id = id
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

	ver := stons.app.versions.getVersion(verSysSingletons)
	switch ver {
	case verUnknown: // no sys.QName storage exists
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

	pKey := toBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletons01))
	return stons.app.storage.Read(context.Background(), pKey, nil, nil, readSingleton)
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
	pKey := toBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletonsLastest))

	batch := make([]istorage.BatchItem, 0)
	for qName, id := range stons.qNames {
		if id >= istructs.FirstSingletonID {
			item := istorage.BatchItem{
				PKey:  pKey,
				CCols: []byte(qName.String()),
				Value: toBytes(uint64(id)),
			}
			batch = append(batch, item)
		}
	}

	if err = stons.app.storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application singleton IDs to storage: %w", err)
	}

	if ver := stons.app.versions.getVersion(verSysSingletons); ver != verSysSingletonsLastest {
		if err = stons.app.versions.putVersion(verSysSingletons, verSysSingletonsLastest); err != nil {
			return fmt.Errorf("error store «sys.Singletons» system view version: %w", err)
		}
	}

	stons.changes = 0
	return nil
}
