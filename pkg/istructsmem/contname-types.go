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

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

// containerNameIDType is identificator for container names
type containerNameIDType uint16

// containerNameCacheType is cache for container name to ID conversions
type containerNameCacheType struct {
	cfg     *AppConfigType
	names   map[string]containerNameIDType
	ids     map[containerNameIDType]string
	lastID  containerNameIDType
	changes uint32
}

func newContainerNameCache(cfg *AppConfigType) containerNameCacheType {
	return containerNameCacheType{
		cfg:    cfg,
		names:  make(map[string]containerNameIDType),
		ids:    make(map[containerNameIDType]string),
		lastID: containerNameIDSysLast,
	}
}

// clear clear QNames cache
func (names *containerNameCacheType) clear() {
	names.names = make(map[string]containerNameIDType)
	names.ids = make(map[containerNameIDType]string)
	names.lastID = containerNameIDSysLast
	names.changes = 0
}

// collectAllContainers retrieves and stores IDs for all known containers in application schemas. Must be called then application starts
func (names *containerNameCacheType) collectAllContainers() (err error) {

	// global constants
	names.collectSysContainer("", nullContainerNameID)
	names.collectSysContainer(istructs.SystemContainer_ViewPartitionKey, viewPKeyContainerID)
	names.collectSysContainer(istructs.SystemContainer_ViewClusteringCols, viewCColsContainerID)
	names.collectSysContainer(istructs.SystemContainer_ViewValue, viewValueContainerID)

	// schemas
	names.cfg.Schemas.EnumSchemas(
		func(schema schemas.Schema) {
			schema.EnumContainers(
				func(c schemas.Container) {
					if !c.IsSys() {
						err = errors.Join(err, names.collectAppContainer(c.Name()))
					}
				})
		})
	return err
}

// collectAppContainer retrieves and stores ID for specified application-level container name
func (names *containerNameCacheType) collectAppContainer(name string) (err error) {
	if _, ok := names.names[name]; ok {
		return nil // already known container
	}

	const maxAvailableID = 0xFFFF

	for id := names.lastID + 1; id < maxAvailableID; id++ {
		if _, ok := names.ids[id]; !ok {
			names.names[name] = id
			names.ids[id] = name
			names.lastID = id
			names.changes++
			return nil
		}
	}

	return ErrContainerNameIDsExceeds
}

// collectSysContainer stores ID for specified system-level container name
func (names *containerNameCacheType) collectSysContainer(qName string, id containerNameIDType) {
	names.names[qName] = id
	names.ids[id] = qName
}

// idToName retrieve container name for specified ID
func (names *containerNameCacheType) idToName(id containerNameIDType) (name string, err error) {
	name, ok := names.ids[id]
	if ok {
		return name, nil
	}

	return "", fmt.Errorf("unknown container name ID «%v»: %w", id, ErrIDNotFound)
}

// load loads all stored container names from storage
func (names *containerNameCacheType) load() (err error) {
	names.clear()

	ver := names.cfg.versions.GetVersion(vers.SysContainersVersion)
	switch ver {
	case vers.UnknownVersion: // no sys.Container storage exists
		return nil
	case verSysContainers01:
		return names.load01()
	}

	return fmt.Errorf("unable load container IDs from «sys.Container» system view version %v: %w", ver, ErrorInvalidVersion)
}

// load01 loads all stored containers from storage using verSysContainers01 codec
func (names *containerNameCacheType) load01() error {

	readName := func(cCols, value []byte) error {
		name := string(cCols)
		if ok, err := schemas.ValidIdent(name); !ok {
			return err
		}

		id := containerNameIDType(binary.BigEndian.Uint16(value))

		names.names[name] = id
		names.ids[id] = name

		if names.lastID < id {
			names.lastID = id
		}

		return nil
	}

	pKey := utils.ToBytes(consts.SysView_Containers, verSysContainers01)
	return names.cfg.storage.Read(context.Background(), pKey, nil, nil, readName)
}

// nameToID retrieve ID for specified container name
func (names *containerNameCacheType) nameToID(name string) (containerNameIDType, error) {
	if id, ok := names.names[name]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("unknown container name «%v»: %w", name, ErrNameNotFound)
}

// prepare loads all container names from storage, add all known system and application container names and store cache if some changes. Must be called at application starts
func (names *containerNameCacheType) prepare() (err error) {
	if err = names.load(); err != nil {
		return err
	}

	if err = names.collectAllContainers(); err != nil {
		return err
	}

	if names.changes > 0 {
		if err := names.store(); err != nil {
			return err
		}
	}

	return nil
}

// store stores all known container names to storage using verSysContainersLastest codec
func (names *containerNameCacheType) store() (err error) {
	pKey := utils.ToBytes(consts.SysView_Containers, verSysContainersLastest)

	batch := make([]istorage.BatchItem, 0)
	for name, id := range names.names {
		if len(name) == 0 || schemas.IsSysContainer(name) {
			continue
		}
		item := istorage.BatchItem{
			PKey:  pKey,
			CCols: []byte(name),
			Value: utils.ToBytes(id),
		}
		batch = append(batch, item)
	}

	if err = names.cfg.storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application containers to storage: %w", err)
	}

	if ver := names.cfg.versions.GetVersion(vers.SysContainersVersion); ver != verSysContainersLastest {
		if err = names.cfg.versions.PutVersion(vers.SysContainersVersion, verSysContainersLastest); err != nil {
			return fmt.Errorf("error store sys.Containers system view version: %w", err)
		}
	}

	names.changes = 0
	return nil
}
