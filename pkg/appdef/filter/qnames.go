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

func (qNamesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_QNames }

func (f qNamesFilter) Match(t appdef.IType) bool {
	return f.names.Contains(t.QName())
}

func (f qNamesFilter) Matches(tt appdef.IWithTypes) appdef.QNames {
	nn := appdef.QNames{}
	for _, n := range f.names {
		if tt.Type(n).Kind() != appdef.TypeKind_null {
			nn.Add(n)
		}
	}
	return nn
}

func (f qNamesFilter) QNames() appdef.QNames { return f.names }

func (f qNamesFilter) String() string {
	return fmt.Sprintf("filter %s %v", f.Kind().TrimString(), f.QNames())
}
