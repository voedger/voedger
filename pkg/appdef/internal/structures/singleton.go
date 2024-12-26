/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.ISingleton
type Singleton struct {
	Doc
	singleton bool
}

// Makes new singleton
func MakeSingleton(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Singleton {
	s := Singleton{
		Doc: MakeDoc(app, ws, name, kind),
	}
	return s
}

func (s *Singleton) Singleton() bool { return s.singleton }

func (s *Singleton) setSingleton() { s.singleton = true }

// # Supports:
//   - appdef.ISingletonBuilder
type SingletonBuilder struct {
	DocBuilder
	*Singleton
}

func MakeSingletonBuilder(singleton *Singleton) SingletonBuilder {
	return SingletonBuilder{
		DocBuilder: MakeDocBuilder(&singleton.Doc),
		Singleton:  singleton,
	}
}

func (sb *SingletonBuilder) SetSingleton() {
	sb.Singleton.setSingleton()
}
