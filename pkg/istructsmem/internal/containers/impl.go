/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func newContainers() *Containers {
	return &Containers{
		containers: make(map[string]ContainerID),
		ids:        make(map[ContainerID]string),
		lastID:     ContainerNameIDSysLast,
	}
}

// Retrieve container for specified ID
func (cnt *Containers) Container(id ContainerID) (name string, err error) {
	name, ok := cnt.ids[id]
	if ok {
		return name, nil
	}

	return "", fmt.Errorf("unknown container ID «%v»: %w", id, ErrContainerIDNotFound)
}

// Retrieve ID for specified container
func (cnt *Containers) ID(name string) (ContainerID, error) {
	if id, ok := cnt.containers[name]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("unknown container name «%v»: %w", name, ErrContainerNotFound)
}

// Loads all container from storage, add all known system and application containers and store if some changes. Must be called at application starts
func (cnt *Containers) Prepare(storage istorage.IAppStorage, versions *vers.Versions, appDef appdef.IAppDef) (err error) {
	if err = cnt.load(storage, versions); err != nil {
		return err
	}

	if err = cnt.collectAll(appDef); err != nil {
		return err
	}

	if cnt.changes > 0 {
		if err := cnt.store(storage, versions); err != nil {
			return err
		}
	}

	return nil
}

// Retrieves and stores IDs for all known containers in application types. Must be called then application starts
func (cnt *Containers) collectAll(appDef appdef.IAppDef) (err error) {

	// system containers
	cnt.collectSys("", NullContainerID)

	// application containers
	if appDef != nil {
		for t := range appDef.Types {
			if cont, ok := t.(appdef.IContainers); ok {
				for _, c := range cont.Containers() {
					err = errors.Join(err, cnt.collect(c.Name()))
				}
			}
		}
	}

	return err
}

// Retrieves and stores ID for specified application container
func (cnt *Containers) collect(name string) (err error) {
	if _, ok := cnt.containers[name]; ok {
		return nil // already known container
	}

	for id := cnt.lastID + 1; id < MaxAvailableContainerID; id++ {
		if _, ok := cnt.ids[id]; !ok {
			cnt.containers[name] = id
			cnt.ids[id] = name
			cnt.lastID = id
			cnt.changes++
			return nil
		}
	}

	return ErrContainerIDsExceeds
}

// Remember ID for specified system container
func (cnt *Containers) collectSys(name string, id ContainerID) {
	cnt.containers[name] = id
	cnt.ids[id] = name
}

// Loads all stored container from storage
func (cnt *Containers) load(storage istorage.IAppStorage, versions *vers.Versions) (err error) {

	ver := versions.Get(vers.SysContainersVersion)
	switch ver {
	case vers.UnknownVersion: // no sys.Container storage exists
		return nil
	case ver01:
		return cnt.load01(storage)
	}

	return fmt.Errorf("unknown version of system Containers view (%v): %w", ver, vers.ErrorInvalidVersion)
}

// Loads all stored containers from storage version ver01
func (cnt *Containers) load01(storage istorage.IAppStorage) error {

	readName := func(cCols, value []byte) error {
		name := string(cCols)
		if ok, err := appdef.ValidIdent(name); !ok {
			return err
		}
		id := ContainerID(binary.BigEndian.Uint16(value))
		if id == NullContainerID {
			return nil // deleted Container
		}

		if id <= ContainerNameIDSysLast {
			return fmt.Errorf("unexpected ID (%v) is loaded from system Containers view: %w", id, ErrWrongContainerID)
		}

		cnt.containers[name] = id
		cnt.ids[id] = name

		if cnt.lastID < id {
			cnt.lastID = id
		}

		return nil
	}

	pKey := utils.ToBytes(consts.SysView_Containers, ver01)
	return storage.Read(context.Background(), pKey, nil, nil, readName)
}

// Stores all known container to storage
func (cnt *Containers) store(storage istorage.IAppStorage, versions *vers.Versions) (err error) {
	pKey := utils.ToBytes(consts.SysView_Containers, latestVersion)

	batch := make([]istorage.BatchItem, 0)
	for name, id := range cnt.containers {
		if name == "" {
			continue // skip NullContainerID
		}
		item := istorage.BatchItem{
			PKey:  pKey,
			CCols: []byte(name),
			Value: utils.ToBytes(id),
		}
		batch = append(batch, item)
	}

	if err = storage.PutBatch(batch); err != nil {
		return fmt.Errorf("error store application container IDs to storage: %w", err)
	}

	if ver := versions.Get(vers.SysContainersVersion); ver != latestVersion {
		if err = versions.Put(vers.SysContainersVersion, latestVersion); err != nil {
			return fmt.Errorf("error store system Containers view version: %w", err)
		}
	}

	cnt.changes = 0
	return nil
}
