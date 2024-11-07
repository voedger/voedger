/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - ISingleton
type singleton struct {
	doc
	singleton bool
}

// Makes new singleton
func makeSingleton(app *appDef, ws *workspace, name QName, kind TypeKind) singleton {
	s := singleton{
		doc: makeDoc(app, ws, name, kind),
	}
	return s
}

func (s *singleton) Singleton() bool {
	return s.singleton
}

func (s *singleton) setSingleton() {
	s.singleton = true
}

// # Implements:
//   - ISingletonBuilder
type singletonBuilder struct {
	docBuilder
	*singleton
}

func makeSingletonBuilder(singleton *singleton) singletonBuilder {
	return singletonBuilder{
		docBuilder: makeDocBuilder(&singleton.doc),
		singleton:  singleton,
	}
}

func (sb *singletonBuilder) SetSingleton() {
	sb.singleton.setSingleton()
}
