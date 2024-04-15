/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Singleton is a document that can be only one instance.
type ISingleton interface {
	IDoc

	// Returns whether the document is a singleton
	Singleton() bool
}

type ISingletonBuilder interface {
	IDocBuilder

	// Устанавливает, что документ является синглтоном
	SetSingleton()
}

type IWithSingletons interface {
	// Return Singleton by name.
	//
	// Returns nil if not found.
	Singleton(QName) ISingleton

	// Enumerates all application singletons
	//
	// Singletons are enumerated in alphabetical order by QName
	Singletons(func(ISingleton))
}
