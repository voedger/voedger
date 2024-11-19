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

// qNamesFilter is a filter that matches types by their qualified names.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type qNamesFilter struct {
	filter
	names appdef.QNames
}

func makeQNamesFilter(name appdef.QName, names ...appdef.QName) appdef.IFilter {
	f := &qNamesFilter{names: appdef.QNamesFrom(name)}
	f.names.Add(names...)
	return f
}

func (qNamesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_QNames }

func (f qNamesFilter) Match(t appdef.IType) bool {
	return f.names.Contains(t.QName())
}

func (f qNamesFilter) QNames() func(func(appdef.QName) bool) {
	return slices.Values(f.names)
}

func (f qNamesFilter) String() string {
	s := fmt.Sprintf("filter.%s(", f.Kind().TrimString())
	for i, c := range f.names {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}
