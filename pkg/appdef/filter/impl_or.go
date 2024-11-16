/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

type orFilter struct {
	filter
	filters []appdef.IFilter
}

func makeOrFilter(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	f := &orFilter{filters: []appdef.IFilter{f1, f2}}
	f.filters = append(f.filters, ff...)
	return f
}

func (orFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Or }

func (f orFilter) Match(t appdef.IType) bool {
	for _, flt := range f.filters {
		if flt.Match(t) {
			return true
		}
	}
	return false
}

func (f orFilter) Matches(tt appdef.IWithTypes) appdef.IWithTypes {
	flt := makeTypes()

	for _, child := range f.filters {
		for t := range child.Matches(tt).Types {
			flt.add(t)
		}
	}

	return flt
}

func (f orFilter) Or() []appdef.IFilter { return f.filters }

func (f orFilter) String() string {
	s := fmt.Sprintf("filter %s(", f.Kind().TrimString())
	for i, flt := range f.Or() {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(flt)
	}
	return s + ")"
}
