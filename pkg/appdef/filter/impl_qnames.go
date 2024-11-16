/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

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
	flt := makeTypes()
	for _, n := range f.names {
		if t := tt.Type(n); t.Kind() != appdef.TypeKind_null {
			flt.add(t)
		}
	}
	return flt
}

func (f qNamesFilter) QNames() appdef.QNames { return f.names }

func (f qNamesFilter) String() string {
	return fmt.Sprintf("filter %s %v", f.Kind().TrimString(), f.QNames())
}
