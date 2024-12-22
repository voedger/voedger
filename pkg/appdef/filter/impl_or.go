/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"iter"
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

func newOrFilter(f1, f2 appdef.IFilter, ff ...appdef.IFilter) *orFilter {
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

func (f orFilter) Or() iter.Seq[appdef.IFilter] { return slices.Values(f.children) }

func (f orFilter) String() string {
	// QNAMES(…) OR TAGS(…)
	// (QNAMES(…) AND TYPES(…)) OR NOT TAGS(…)
	s := ""
	for i, c := range f.children {
		cStr := fmt.Sprint(c)
		if (c.Kind() == appdef.FilterKind_Or) || (c.Kind() == appdef.FilterKind_And) {
			cStr = fmt.Sprintf("(%s)", cStr)
		}
		if i > 0 {
			s += " OR "
		}
		s += cStr
	}
	return s
}
