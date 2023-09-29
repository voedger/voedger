/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
)

func newSchemes() *DynoBufSchemes {
	cache := &DynoBufSchemes{
		schemes: make(map[string]*dynobuffers.Scheme),
	}
	return cache
}

// Prepares schemes
func (sch *DynoBufSchemes) Prepare(appDef appdef.IAppDef) {
	appDef.Types(
		func(t appdef.IType) {
			if view, ok := t.(appdef.IView); ok {
				sch.addView(view)
				return
			}
			if fld, ok := t.(appdef.IFields); ok {
				sch.add(t.QName().String(), fld)
			}
		})
}

// Returns structure scheme. Nil if not found
//
// This method can be used to get scheme for:
//   - any structured type (doc or record),
//   - view value
func (sch DynoBufSchemes) Scheme(name appdef.QName) *dynobuffers.Scheme {
	return sch.schemes[name.String()]
}

// Returns view key scheme. Nil if not found
//
// TODO: check this method is used?
func (sch DynoBufSchemes) ViewKeyScheme(name appdef.QName) *dynobuffers.Scheme {
	return sch.schemes[name.String()+viewKeySuffix]
}

// Returns view partition key scheme. Nil if not found
func (sch DynoBufSchemes) ViewPartKeyScheme(name appdef.QName) *dynobuffers.Scheme {
	return sch.schemes[name.String()+viewPartKeySuffix]
}

// Returns view clustering columns scheme. Nil if not found
func (sch DynoBufSchemes) ViewClustColsScheme(name appdef.QName) *dynobuffers.Scheme {
	return sch.schemes[name.String()+viewClustColsSuffix]
}

// Adds scheme
func (sch *DynoBufSchemes) add(name string, fields appdef.IFields) {
	sch.schemes[name] = NewFieldsScheme(name, fields)
}

// Adds four view schemes:
//   - view key,
//   - partition key,
//   - clustering columns and
//   - view value
func (sch *DynoBufSchemes) addView(view appdef.IView) {
	name := view.QName().String()
	sch.add(name+viewKeySuffix, view.Key())
	sch.add(name+viewPartKeySuffix, view.Key().Partition())
	sch.add(name+viewClustColsSuffix, view.Key().ClustCols())
	sch.add(name, view.Value())
}
