/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//	- IExtension
type extension struct {
	name   string
	engine ExtensionEngineKind
}

func (ex *extension) Name() string {
	return ex.name
}

func (ex *extension) Engine() ExtensionEngineKind {
	return ex.engine
}
