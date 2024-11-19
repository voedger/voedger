/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

// notFilter realizes filter negotiation.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type notFilter struct {
	filter
	f appdef.IFilter
}

func makeNotFilter(f appdef.IFilter) appdef.IFilter {
	return &notFilter{f: f}
}

func (notFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Not }

func (f notFilter) Match(t appdef.IType) bool {
	return !f.Not().Match(t)
}

func (f notFilter) Not() appdef.IFilter { return f.f }

func (f notFilter) String() string {
	return fmt.Sprintf("filter.%s(%v)", f.Kind().TrimString(), f.Not())
}
