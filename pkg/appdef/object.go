/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//   - IObject, IObjectBuilder
type object struct {
	structure
}

func newObject(app *appDef, name QName) *object {
	o := &object{}
	o.structure = makeStructure(app, name, TypeKind_Object, o)
	app.appendType(o)
	return o
}

func (o *object) isObject() {}

type objRef struct {
	name QName
	obj  IObject
}

// Returns object by reference
func (o *objRef) object(app IAppDef) IObject {
	if (o.name == NullQName) || (o.name == QNameANY) {
		return nil
	}
	if (o.obj == nil) || (o.obj.QName() != o.name) {
		o.obj = app.Object(o.name)
	}
	return o.obj
}

// Sets reference name
func (o *objRef) setName(n QName) {
	o.name = n
	o.obj = nil
}

// Returns is reference valid
func (o *objRef) valid(app IAppDef) (bool, error) {
	if (o.name == NullQName) || (o.name == QNameANY) || (o.object(app) != nil) {
		return true, nil
	}
	return false, fmt.Errorf("object type «%v» is not found: %w", o.name, ErrNameNotFound)
}
