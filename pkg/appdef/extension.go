/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//	- IExtension
type extension struct {
	comment
	name   string
	engine ExtensionEngineKind
}

func newExtension() *extension {
	return &extension{}
}

func (ex extension) Name() string {
	return ex.name
}

func (ex extension) Engine() ExtensionEngineKind {
	return ex.engine
}

func (ex extension) String() string {
	return fmt.Sprintf("%s (%s)", ex.Name(), ex.Engine().TrimString())
}
