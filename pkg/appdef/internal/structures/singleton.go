/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.ISingleton
type SingletonDoc struct {
	Doc
	singleton bool
}

// Makes new singleton
func MakeSingleton(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) SingletonDoc {
	return SingletonDoc{Doc: MakeDoc(ws, name, kind)}
}

func (s SingletonDoc) Singleton() bool { return s.singleton }

func (s *SingletonDoc) setSingleton() { s.singleton = true }

// # Supports:
//   - appdef.ISingletonBuilder
type SingletonBuilder struct {
	DocBuilder
	*SingletonDoc
}

func MakeSingletonBuilder(singleton *SingletonDoc) SingletonBuilder {
	return SingletonBuilder{
		DocBuilder:   MakeDocBuilder(&singleton.Doc),
		SingletonDoc: singleton,
	}
}

func (sb *SingletonBuilder) SetSingleton() {
	sb.setSingleton()
}
