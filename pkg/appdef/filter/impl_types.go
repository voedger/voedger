/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"iter"

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
	ws    appdef.QName
	types appdef.TypeKindSet
}

func makeTypesFilter(ws appdef.QName, t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	f := &typesFilter{ws: ws, types: set.From(t)}
	f.types.Set(tt...)
	return f
}

func (typesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Types }

func (f typesFilter) Match(t appdef.IType) bool {
	return ((f.ws == appdef.NullQName) || (t.Workspace().QName() == f.ws)) &&
		f.types.Contains(t.Kind())
}

func (f typesFilter) String() string {
	s := ""
	for t := range f.types.Values() {
		if len(s) > 0 {
			s += ", "
		}
		s += t.TrimString()
	}
	if f.ws != appdef.NullQName {
		s = fmt.Sprintf("workspace «%s»: %s", f.ws, s)
	}
	return fmt.Sprintf("filter.%s(%s)", f.Kind().TrimString(), s)
}

func (f typesFilter) Types() iter.Seq[appdef.TypeKind] { return f.types.Values() }
