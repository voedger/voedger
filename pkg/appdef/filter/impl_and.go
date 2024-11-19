/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"slices"

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

func (f andFilter) And() func(func(appdef.IFilter) bool) { return slices.Values(f.children) }

func (andFilter) Kind() appdef.FilterKind { return appdef.FilterKind_And }

func (f andFilter) Match(t appdef.IType) bool {
	for c := range f.And() {
		if !c.Match(t) {
			return false
		}
	}
	return true
}

func (f andFilter) String() string {
	s := fmt.Sprintf("filter.%s(", f.Kind().TrimString())
	for i, c := range f.children {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}
