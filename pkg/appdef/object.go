/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IObject, IObjectBuilder
type object struct {
	typ
	comment
	fields
	containers
	withAbstract
}

func newObject(app *appDef, name QName) *object {
	obj := &object{
		typ: makeType(app, name, TypeKind_Object),
	}
	obj.fields = makeFields(obj)
	obj.containers = makeContainers(obj)
	app.appendType(obj)
	return obj
}

// # Implements:
//   - IElement, IElementBuilder
type element struct {
	typ
	comment
	fields
	containers
	withAbstract
}

func newElement(app *appDef, name QName) *element {
	elt := &element{
		typ: makeType(app, name, TypeKind_Element),
	}
	elt.fields = makeFields(elt)
	elt.containers = makeContainers(elt)
	app.appendType(elt)
	return elt
}

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
