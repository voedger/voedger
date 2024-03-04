/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// # Implements:
//   - IStorage
type storage struct {
	comment
	QName
	names QNames
}

func newStorage(name QName, names ...QName) *storage {
	return &storage{
		comment: makeComment(),
		QName:   name,
		names:   QNamesFrom(names...),
	}
}

func (s *storage) Name() QName   { return s.QName }
func (s *storage) Names() QNames { return s.names }

func (s *storage) String() string {
	return fmt.Sprintf("Storage «%v» %v", s.QName, s.names)
}

// # Implements:
//   - IStorages & IStoragesBuilder
type storages struct {
	emb      interface{}
	storages map[QName]*storage
	qnames   map[QName]QNames
	ordered  QNames
}

func newStorages(embeds interface{}) *storages {
	return &storages{
		emb:      embeds,
		storages: make(map[QName]*storage),
		qnames:   make(map[QName]QNames),
		ordered:  make(QNames, 0),
	}
}

func (ss *storages) Add(name QName, names ...QName) IStoragesBuilder {
	if name == NullQName {
		panic(fmt.Errorf("%v: empty storage name: %w", ss.embeds(), ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("%v: invalid storage name «%v»: %w", ss.embeds(), name, err))
	}
	if ok, err := ValidQNames(names...); !ok {
		panic(fmt.Errorf("%v: invalid names for storage «%v»: %w", ss.embeds(), name, err))
	}
	s, ok := ss.storages[name]
	if ok {
		s.names.Add(names...)
	} else {
		s = newStorage(name, names...)
		ss.storages[name] = s
		ss.ordered.Add(name)
	}
	ss.qnames[name] = s.names
	return ss
}

func (ss *storages) Map() map[QName]QNames { return ss.qnames }

func (ss *storages) Enum(cb func(IStorage)) {
	for _, n := range ss.ordered {
		cb(ss.storages[n])
	}
}

func (ss *storages) SetComment(name QName, comment string) IStoragesBuilder {
	if s, ok := ss.storages[name]; ok {
		s.SetComment(comment)
		return ss
	}
	panic(fmt.Errorf("%v: storage «%v» not found: %w", ss.embeds(), name, ErrNameNotFound))
}

func (ss *storages) Storage(name QName) IStorage {
	if s, ok := ss.storages[name]; ok {
		return s
	}
	return nil
}

// Returns type that embeds storages
func (ss *storages) embeds() IType {
	return ss.emb.(IType)
}
