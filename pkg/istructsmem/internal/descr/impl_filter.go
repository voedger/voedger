/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
)

func newFilter() *Filter {
	return &Filter{}
}

func (f *Filter) read(flt appdef.IFilter) {
	switch flt.Kind() {
	case appdef.FilterKind_QNames:
		f.QNames = slices.Collect(flt.QNames())
	case appdef.FilterKind_Types:
		f.Types = slices.Collect(flt.Types())
	case appdef.FilterKind_Tags:
		f.Tags = slices.Collect(flt.Tags())
	case appdef.FilterKind_And:
		for cf := range flt.And() {
			c := newFilter()
			c.read(cf)
			f.And = append(f.And, c)
		}
	case appdef.FilterKind_Or:
		for cf := range flt.Or() {
			c := newFilter()
			c.read(cf)
			f.Or = append(f.Or, c)
		}
	case appdef.FilterKind_Not:
		f.Not = newFilter()
		f.Not.read(flt.Not())
	default:
		// notest: fullcase switch
		panic("Unknown filter kind " + flt.Kind().String())
	}
}
