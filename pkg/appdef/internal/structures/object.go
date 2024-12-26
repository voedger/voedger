/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.IObject
type Object struct {
	Structure
}

func NewObject(ws appdef.IWorkspace, name appdef.QName) *Object {
	return &Object{Structure: MakeStructure(ws, name, appdef.TypeKind_Object)}
}

// # Supports:
//   - appdef.IObjectBuilder
type ObjectBuilder struct {
	StructureBuilder
	*Object
}

func NewObjectBuilder(o *Object) *ObjectBuilder {
	return &ObjectBuilder{
		StructureBuilder: MakeStructureBuilder(&o.Structure),
		Object:           o,
	}
}
