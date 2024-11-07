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

	// Makes the doc a singleton
	SetSingleton()
}
