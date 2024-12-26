/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
)

// # Supports:
//   - appdef.IStorage
type Storage struct {
	comments.WithComments
	appdef.QName
	app   appdef.IAppDef
	names appdef.QNames
}

func NewStorage(app appdef.IAppDef, name appdef.QName, names ...appdef.QName) *Storage {
	return &Storage{
		WithComments: comments.MakeWithComments(),
		QName:        name,
		app:          app,
		names:        appdef.QNamesFrom(names...),
	}
}

func (s Storage) Name() appdef.QName   { return s.QName }
func (s Storage) Names() appdef.QNames { return s.names }

func (s Storage) String() string {
	return fmt.Sprintf("Storage «%v» %v", s.QName, s.names)
}

func (s Storage) Validate() (err error) {
	for _, n := range s.names {
		if s.app.Type(n).Kind() == appdef.TypeKind_null {
			err = errors.Join(err,
				appdef.ErrNotFound("storage «%v» type «%v»", s.QName, n))
			break
		}
	}
	return err
}

// # Supports:
//   - appdef.IStorages
type Storages struct {
	app      appdef.IAppDef
	storages map[appdef.QName]*Storage
	qnames   map[appdef.QName]appdef.QNames
	ordered  appdef.QNames
}

func NewStorages(app appdef.IAppDef) *Storages {
	return &Storages{
		app:      app,
		storages: make(map[appdef.QName]*Storage),
		qnames:   make(map[appdef.QName]appdef.QNames),
		ordered:  make(appdef.QNames, 0),
	}
}

func (ss Storages) Len() int { return len(ss.storages) }

func (ss Storages) Map() map[appdef.QName]appdef.QNames { return ss.qnames }

func (ss Storages) Enum(cb func(appdef.IStorage) bool) {
	for _, n := range ss.ordered {
		if !cb(ss.storages[n]) {
			break
		}
	}
}

func (ss Storages) Storage(name appdef.QName) appdef.IStorage {
	if s, ok := ss.storages[name]; ok {
		return s
	}
	return nil
}

func (ss *Storages) add(name appdef.QName, names ...appdef.QName) {
	if name == appdef.NullQName {
		panic(appdef.ErrMissed("storage name"))
	}
	if ok, err := appdef.ValidQName(name); !ok {
		panic(fmt.Errorf("invalid storage name «%v»: %w", name, err))
	}
	if ok, err := appdef.ValidQNames(names...); !ok {
		panic(fmt.Errorf("invalid names for storage «%v»: %w", name, err))
	}
	s, ok := ss.storages[name]
	if ok {
		s.names.Add(names...)
	} else {
		s = NewStorage(ss.app, name, names...)
		ss.storages[name] = s
		ss.ordered.Add(name)
	}
	ss.qnames[name] = s.names
}

func (ss *Storages) setComment(name appdef.QName, comment string) {
	if s, ok := ss.storages[name]; ok {
		comments.SetComment(s.WithComments, comment)
		return
	}
	panic(appdef.ErrNotFound("storage «%v»", name))
}

func (ss Storages) Validate() (err error) {
	for _, s := range ss.storages {
		err = errors.Join(err, s.Validate())
	}
	return err
}

// # Supports:
//   - appdef.IStoragesBuilder
type StoragesBuilder struct {
	*Storages
}

func NewStoragesBuilder(storages *Storages) *StoragesBuilder {
	return &StoragesBuilder{storages}
}

func (sb *StoragesBuilder) Add(name appdef.QName, names ...appdef.QName) appdef.IStoragesBuilder {
	sb.Storages.add(name, names...)
	return sb
}

func (sb *StoragesBuilder) SetComment(name appdef.QName, comment string) appdef.IStoragesBuilder {
	sb.Storages.setComment(name, comment)
	return sb
}
