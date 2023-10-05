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

func (o *object) IsObject() bool { return true }

// # Implements:
//   - IElement, IElementBuilder
type element struct {
	structure
}

func newElement(app *appDef, name QName) *element {
	e := &element{}
	e.structure = makeStructure(app, name, TypeKind_Element, e)
	app.appendType(e)
	return e
}

func (e *element) IsElement() bool { return true }

type objRef struct {
	name QName
	obj  IObject
}

func (o *objRef) object(app IAppDef) IObject {
	if o.name == NullQName {
		return nil
	}
	if (o.obj == nil) || (o.obj.QName() != o.name) {
		o.obj = app.Object(o.name)
	}
	return o.obj
}

func (o *objRef) setName(n QName) {
	o.name = n
	o.obj = nil
}
