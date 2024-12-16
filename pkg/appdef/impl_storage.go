/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - IStorage
type storage struct {
	comment
	QName
	app   *appDef
	names QNames
}

func newStorage(app *appDef, name QName, names ...QName) *storage {
	return &storage{
		comment: makeComment(),
		QName:   name,
		app:     app,
		names:   QNamesFrom(names...),
	}
}

func (s *storage) Name() QName   { return s.QName }
func (s *storage) Names() QNames { return s.names }

func (s *storage) String() string {
	return fmt.Sprintf("Storage «%v» %v", s.QName, s.names)
}

func (s storage) validate() (err error) {
	for _, n := range s.names {
		if s.app.Type(n).Kind() == TypeKind_null {
			err = errors.Join(err,
				ErrNotFound("storage «%v» type «%v»", s.QName, n))
			break
		}
	}
	return err
}

// # Implements:
//   - IStorages
type storages struct {
	app      *appDef
	storages map[QName]*storage
	qnames   map[QName]QNames
	ordered  QNames
}

func newStorages(app *appDef) *storages {
	return &storages{
		app:      app,
		storages: make(map[QName]*storage),
		qnames:   make(map[QName]QNames),
		ordered:  make(QNames, 0),
	}
}

func (ss *storages) Len() int { return len(ss.storages) }

func (ss *storages) Map() map[QName]QNames { return ss.qnames }

func (ss *storages) Enum(cb func(IStorage) bool) {
	for _, n := range ss.ordered {
		if !cb(ss.storages[n]) {
			break
		}
	}
}

func (ss *storages) Storage(name QName) IStorage {
	if s, ok := ss.storages[name]; ok {
		return s
	}
	return nil
}

func (ss *storages) add(name QName, names ...QName) {
	if name == NullQName {
		panic(ErrMissed("storage name"))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("invalid storage name «%v»: %w", name, err))
	}
	if ok, err := ValidQNames(names...); !ok {
		panic(fmt.Errorf("invalid names for storage «%v»: %w", name, err))
	}
	s, ok := ss.storages[name]
	if ok {
		s.names.Add(names...)
	} else {
		s = newStorage(ss.app, name, names...)
		ss.storages[name] = s
		ss.ordered.Add(name)
	}
	ss.qnames[name] = s.names
}

func (ss *storages) setComment(name QName, comment string) {
	if s, ok := ss.storages[name]; ok {
		s.comment.setComment(comment)
		return
	}
	panic(ErrNotFound("storage «%v»", name))
}

func (ss storages) validate() (err error) {
	for _, s := range ss.storages {
		err = errors.Join(err, s.validate())
	}
	return err
}

// # Implements:
//   - IStoragesBuilder
type storagesBuilder struct {
	storages *storages
}

func newStoragesBuilder(storages *storages) *storagesBuilder {
	return &storagesBuilder{
		storages: storages,
	}
}

func (sb *storagesBuilder) Add(name QName, names ...QName) IStoragesBuilder {
	sb.storages.add(name, names...)
	return sb
}

func (sb *storagesBuilder) SetComment(name QName, comment string) IStoragesBuilder {
	sb.storages.setComment(name, comment)
	return sb
}
