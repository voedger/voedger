/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// QNameID is identificator for QNames
type QNameID uint16

// qNameCacheType is cache for QName IDs conversions
type qNameCacheType struct {
	app     *AppConfigType
	qNames  map[istructs.QName]QNameID
	ids     map[QNameID]istructs.QName
	lastID  QNameID
	changes uint32
}

func newQNameCache(app *AppConfigType) qNameCacheType {
	return qNameCacheType{
		app:    app,
		qNames: make(map[istructs.QName]QNameID),
		ids:    make(map[QNameID]istructs.QName),
		lastID: QNameIDSysLast,
	}
}

// clear clear QNames cache
func (names *qNameCacheType) clear() {
	names.qNames = make(map[istructs.QName]QNameID)
	names.ids = make(map[QNameID]istructs.QName)
	names.lastID = QNameIDSysLast
	names.changes = 0
}

// collectAllQNames collect all system and application QName IDs
func (names *qNameCacheType) collectAllQNames() (err error) {

	// system QNames
	names.
		collectSysQName(istructs.NullQName, NullQNameID).
		collectSysQName(istructs.QNameForError, QNameIDForError).
		collectSysQName(istructs.QNameCommandCUD, QNameIDCommandCUD)

	// schemas
	for qn := range names.app.Schemas.schemas {
		if err = names.collectAppQName(qn); err != nil {
			return err
		}
	}
	// resources
	for qn := range names.app.Resources.resources {
		if err = names.collectAppQName(qn); err != nil {
			return err
		}
	}

	return nil
}

// collectAppQName checks is exists ID for application QName in cache. If not then adds it with new ID
func (names *qNameCacheType) collectAppQName(qName istructs.QName) (err error) {
	if _, ok := names.qNames[qName]; ok {
		return nil // already known QName
	}

	const maxAvailableID = 0xFFFF

	for id := names.lastID + 1; id < maxAvailableID; id++ {
		if _, ok := names.ids[id]; !ok {
			names.qNames[qName] = id
			names.ids[id] = qName
			names.lastID = id
			names.changes++
			return nil
		}
	}

	return ErrQNameIDsExceeds
}

// collectQName adds system QName to cache
func (names *qNameCacheType) collectSysQName(qName istructs.QName, id QNameID) *qNameCacheType {
	names.qNames[qName] = id
	names.ids[id] = qName
	return names
}

// idToQName retrieve QName for specified ID
func (names *qNameCacheType) idToQName(id QNameID) (qName istructs.QName, err error) {
	qName, ok := names.ids[id]
	if ok {
		return qName, nil
	}

	return istructs.NullQName, fmt.Errorf("unknown QName ID «%v»: %w", id, ErrIDNotFound)
}

// load loads all stored QNames from storage
func (names *qNameCacheType) load() (err error) {
	names.clear()

	ver := names.app.versions.getVersion(verSysQNames)
	switch ver {
	case verUnknown: // no sys.QName storage exists
		return nil
	case verSysQNames01:
		return names.load01()
	}

	return fmt.Errorf("unable load application QName IDs from «sys.QNames» system view version %v: %w", ver, ErrorInvalidVersion)
}

// load01 loads all stored QNames from storage using verSysQNames01 codec
func (names *qNameCacheType) load01() error {

	readQName := func(cCols, value []byte) error {
		qName, err := istructs.ParseQName(string(cCols))
		if err != nil {
			return err
		}
		id := QNameID(binary.BigEndian.Uint16(value))
		if id == NullQNameID {
			return nil // deleted QName
		}

		names.qNames[qName] = id
		names.ids[id] = qName

		if names.lastID < id {
			names.lastID = id
		}

		return nil
	}
	pKey := toBytes(uint16(QNameIDSysQNames), uint16(verSysQNames01))
	return names.app.storage.Read(context.Background(), pKey, nil, nil, readQName)
}

// prepare loads all QNames from storage, add all known system and application QName IDs and store cache if some changes. Must be called at application starts
func (names *qNameCacheType) prepare() (err error) {
	if err = names.load(); err != nil {
		return err
	}

	if err = names.collectAllQNames(); err != nil {
		return err
	}

	if names.changes > 0 {
		if err := names.store(); err != nil {
			return err
		}
	}

	return nil
}

// qNameToID retrieve ID for specified QName
func (names *qNameCacheType) qNameToID(qName istructs.QName) (QNameID, error) {
	if id, ok := names.qNames[qName]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("unknown QName «%v»: %w", qName, ErrNameNotFound)
}

// store stores all known QNames to storage using verSysQNamesLastest codec
func (names *qNameCacheType) store() (err error) {
	pKey := toBytes(uint16(QNameIDSysQNames), uint16(verSysQNamesLastest))

	batch := make([]istorage.BatchItem, 0)
	for qName, id := range names.qNames {
		if id > QNameIDSysLast {
			item := istorage.BatchItem{
				PKey:  pKey,
				CCols: []byte(qName.String()),
				Value: toBytes(uint16(id)),
			}
			batch = append(batch, item)
		}
	}

	if err = names.app.storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application QName IDs to storage: %w", err)
	}

	if ver := names.app.versions.getVersion(verSysQNames); ver != verSysQNamesLastest {
		if err = names.app.versions.putVersion(verSysQNames, verSysQNamesLastest); err != nil {
			return fmt.Errorf("error store «sys.QNames» system view version: %w", err)
		}
	}

	names.changes = 0
	return nil
}
