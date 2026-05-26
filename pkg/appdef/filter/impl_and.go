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

// orFilter realizes filter conjunction.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type andFilter struct {
	filter
	children []appdef.IFilter
}

func newAndFilter(ff ...appdef.IFilter) *andFilter {
	if len(ff) < 1+1 {
		panic("less then two filters are provided")
	}
	return &andFilter{children: slices.Clone(ff)}
}

func (f andFilter) And() []appdef.IFilter { return f.children }

func (andFilter) Kind() appdef.FilterKind { return appdef.FilterKind_And }

func (f andFilter) Match(t appdef.IType) bool {
	for _, c := range f.And() {
		if !c.Match(t) {
			return false
		}
	}
	return true
}

func (f andFilter) String() string {
	// QNAMES(…) AND TAGS(…)
	// (QNAMES(…) OR TYPES(…)) AND NOT TAGS(…)
	parts := make([]string, len(f.children))
	for i, c := range f.children {
		cStr := fmt.Sprint(c)
		if (c.Kind() == appdef.FilterKind_Or) || (c.Kind() == appdef.FilterKind_And) {
			cStr = fmt.Sprintf("(%s)", cStr)
		}
		parts[i] = cStr
	}
	return strings.Join(parts, " AND ")
}
