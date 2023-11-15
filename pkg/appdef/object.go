/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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
