/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - ISingleton, ISingletonBuilder

type singleton struct {
	doc
	singleton bool
}

// Makes new singleton
func makeSingleton(app *appDef, name QName, kind TypeKind, parent interface{}) singleton {
	s := singleton{}
	s.doc = makeDoc(app, name, kind, parent)
	return s
}

func (s *singleton) SetSingleton() {
	s.singleton = true
}

func (s *singleton) Singleton() bool {
	return s.singleton
}
