/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// typesFilter is a filter that matches types by their kind.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type typesFilter struct {
	filter
	types appdef.TypeKindSet
}

func makeTypesFilter(t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	f := &typesFilter{types: set.From(t)}
	f.types.Set(tt...)
	return f
}

func (typesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Types }

func (f typesFilter) Match(t appdef.IType) bool {
	return f.types.Contains(t.Kind())
}

func (f typesFilter) String() string {
	s := ""
	for t := range f.types.All {
		if len(s) > 0 {
			s += ", "
		}
		s += t.TrimString()
	}
	return fmt.Sprintf("filter.%s(%s)", f.Kind().TrimString(), s)
}

func (f typesFilter) Types() func(func(appdef.TypeKind) bool) { return f.types.All }
