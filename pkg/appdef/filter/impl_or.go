/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"slices"
	"strings"

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

func newOrFilter(ff ...appdef.IFilter) *orFilter {
	if len(ff) < 1+1 {
		panic("less then two filters are provided")
	}
	return &orFilter{children: slices.Clone(ff)}
}

func (orFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Or }

func (f orFilter) Match(t appdef.IType) bool {
	for _, c := range f.Or() {
		if c.Match(t) {
			return true
		}
	}
	return false
}

func (f orFilter) Or() []appdef.IFilter { return f.children }

func (f orFilter) String() string {
	// QNAMES(…) OR TAGS(…)
	// (QNAMES(…) AND TYPES(…)) OR NOT TAGS(…)
	parts := make([]string, len(f.children))
	for i, c := range f.children {
		cStr := fmt.Sprint(c)
		if (c.Kind() == appdef.FilterKind_Or) || (c.Kind() == appdef.FilterKind_And) {
			cStr = fmt.Sprintf("(%s)", cStr)
		}
		parts[i] = cStr
	}
	return strings.Join(parts, " OR ")
}
