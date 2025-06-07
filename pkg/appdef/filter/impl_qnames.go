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

func newQNamesFilter(names ...appdef.QName) *qNamesFilter {
	if len(names) == 0 {
		panic("no qualified names specified")
	}
	return &qNamesFilter{names: appdef.QNamesFrom(names...)}
}

func (qNamesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_QNames }

func (f qNamesFilter) Match(t appdef.IType) bool {
	return f.names.Contains(t.QName())
}

func (f qNamesFilter) QNames() []appdef.QName {
	return f.names
}

func (f qNamesFilter) String() string {
	// QNAMES(…)
	s := "QNAMES("
	for i, c := range f.names {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}
