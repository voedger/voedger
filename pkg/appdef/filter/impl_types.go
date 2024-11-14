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

type typesFilter struct {
	filter
	types appdef.TypeKindSet
}

func types(t appdef.TypeKind, tt ...appdef.TypeKind) IFilter {
	f := &typesFilter{types: set.From(t)}
	f.types.Set(tt...)
	return f
}

func (typesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Types }

func (f typesFilter) Match(t appdef.IType) bool {
	return f.types.Contains(t.Kind())
}

func (f typesFilter) Matches(tt appdef.IWithTypes) appdef.QNames {
	nn := appdef.QNames{}
	for t := range tt.Types {
		if f.types.Contains(t.Kind()) {
			nn.Add(t.QName())
		}
	}
	return nn
}

func (f typesFilter) String() string {
	return fmt.Sprintf("filter %s %v", f.Kind().TrimString(), f.Types())
}

func (f typesFilter) Types() appdef.TypeKindSet { return f.types }
