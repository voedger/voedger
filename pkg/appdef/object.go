/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IObject, IObjectBuilder
type object struct {
	def
	fields
	containers
}

func newObject(app *appDef, name QName) *object {
	obj := &object{
		def: makeDef(app, name, DefKind_Object),
	}
	obj.fields = makeFields(obj)
	obj.containers = makeContainers(obj)
	app.appendDef(obj)
	return obj
}

// # Implements:
//   - IElement, IElementBuilder
type element struct {
	def
	fields
	containers
}

func newElement(app *appDef, name QName) *element {
	elt := &element{
		def: makeDef(app, name, DefKind_Element),
	}
	elt.fields = makeFields(elt)
	elt.containers = makeContainers(elt)
	app.appendDef(elt)
	return elt
}

type objRef struct {
	name QName
	def  IObject
}

func (o *objRef) object(app IAppDef) IObject {
	if o.name == NullQName {
		return nil
	}
	if (o.def == nil) || (o.def.QName() != o.name) {
		o.def = app.Object(o.name)
	}
	return o.def
}

func (o *objRef) setName(n QName) {
	o.name = n
	o.def = nil
}
