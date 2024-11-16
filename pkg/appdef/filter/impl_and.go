/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

type andFilter struct {
	filter
	filters []appdef.IFilter
}

func makeAndFilter(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	f := &andFilter{filters: []appdef.IFilter{f1, f2}}
	f.filters = append(f.filters, ff...)
	return f
}

func (f andFilter) And() []appdef.IFilter { return f.filters }

func (andFilter) Kind() appdef.FilterKind { return appdef.FilterKind_And }

func (f andFilter) Match(t appdef.IType) bool {
	for _, flt := range f.filters {
		if !flt.Match(t) {
			return false
		}
	}
	return true
}

func (f andFilter) Matches(tt appdef.IWithTypes) appdef.IWithTypes {
	var flt appdef.IWithTypes = cloneTypes(tt)

	for _, child := range f.filters {
		flt = child.Matches(flt)
		if flt.TypeCount() == 0 {
			break
		}
	}

	return flt
}

func (f andFilter) String() string {
	s := fmt.Sprintf("filter %s(", f.Kind().TrimString())
	for i, flt := range f.And() {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(flt)
	}
	return s + ")"
}
