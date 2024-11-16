/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

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

func (f qNamesFilter) Matches(tt appdef.IWithTypes) appdef.IWithTypes {
	r := makeResults()
	for _, n := range f.names {
		if t := tt.Type(n); t.Kind() != appdef.TypeKind_null {
			r.add(t)
		}
	}
	return r
}

func (f qNamesFilter) QNames() appdef.QNames { return f.names }

func (f qNamesFilter) String() string {
	return fmt.Sprintf("filter %s %v", f.Kind().TrimString(), f.QNames())
}
