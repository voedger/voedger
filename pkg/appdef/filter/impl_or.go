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

// orFilter realizes filter disjunction.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type orFilter struct {
	filter
	children []appdef.IFilter
}

func makeOrFilter(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	f := &orFilter{children: []appdef.IFilter{f1, f2}}
	f.children = append(f.children, ff...)
	return f
}

func (orFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Or }

func (f orFilter) Match(t appdef.IType) bool {
	for c := range f.Or() {
		if c.Match(t) {
			return true
		}
	}
	return false
}

func (f orFilter) Or() func(func(appdef.IFilter) bool) { return slices.Values(f.children) }

func (f orFilter) String() string {
	s := fmt.Sprintf("filter.%s(", f.Kind().TrimString())
	for i, c := range f.children {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}
