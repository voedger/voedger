/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "iter"

type IStorage interface {
	IWithComments

	// Returns storage name.
	Name() QName

	// Returns names in storage in alphabetical order.
	Names() iter.Seq[QName]
}

type IStorages interface {
	// Returns storage by name.
	//
	// Returns nil if storage not found.
	Storage(name QName) IStorage

	// Returns storages in alphabetical order.
	All() iter.Seq2[QName, IStorage]
}

type IStoragesBuilder interface {
	// Add storage.
	//
	// If storage with name is already exists in states then names will be added to existing storage.
	//
	// # Panics:
	//	- if name is empty,
	//	- if name is invalid,
	//	- if names contains invalid name(s).
	Add(name QName, names ...QName) IStoragesBuilder

	// Sets comment for storage.
	//
	// # Panics:
	//	- if storage with name is not added.
	SetComment(name QName, comment string) IStoragesBuilder
}
