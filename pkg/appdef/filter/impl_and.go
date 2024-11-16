/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

// orFilter realizes filter conjunction.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type andFilter struct {
	filter
	children []appdef.IFilter
}

func makeAndFilter(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	f := &andFilter{children: []appdef.IFilter{f1, f2}}
	f.children = append(f.children, ff...)
	return f
}

func (f andFilter) And() []appdef.IFilter { return f.children }

func (andFilter) Kind() appdef.FilterKind { return appdef.FilterKind_And }

func (f andFilter) Match(t appdef.IType) bool {
	for _, c := range f.children {
		if !c.Match(t) {
			return false
		}
	}
	return true
}

func (f andFilter) Matches(tt appdef.IWithTypes) appdef.IWithTypes {
	var r appdef.IWithTypes = copyResults(tt)

	for _, child := range f.children {
		r = child.Matches(r)
		if r.TypeCount() == 0 {
			break
		}
	}

	return r
}

func (f andFilter) String() string {
	s := fmt.Sprintf("filter %s(", f.Kind().TrimString())
	for i, c := range f.And() {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}
