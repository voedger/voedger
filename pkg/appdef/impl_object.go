/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IObject
type object struct {
	structure
}

func newObject(app *appDef, name QName) *object {
	o := &object{}
	o.structure = makeStructure(app, name, TypeKind_Object)
	app.appendType(o)
	return o
}

func (o *object) isObject() {}

// # Implements:
//   - IObjectBuilder
type objectBuilder struct {
	structureBuilder
	*object
}

func newObjectBuilder(object *object) *objectBuilder {
	return &objectBuilder{
		structureBuilder: makeStructureBuilder(&object.structure),
		object:           object,
	}
}
