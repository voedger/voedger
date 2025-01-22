/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

// tagsFilter is a filter that matches types by their tags.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type tagsFilter struct {
	filter
	tags appdef.QNames
}

func newTagsFilter(tags ...appdef.QName) *tagsFilter {
	if len(tags) == 0 {
		panic("no tags provided")
	}
	return &tagsFilter{tags: appdef.QNamesFrom(tags...)}
}

func (tagsFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Tags }

func (f tagsFilter) Match(t appdef.IType) bool {
	for _, tag := range f.tags {
		if t.HasTag(tag) {
			return true
		}
	}
	return false
}

func (f tagsFilter) String() string {
	// TAGS(…)
	s := "TAGS("
	for i, c := range f.tags {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}

func (f tagsFilter) Tags() []appdef.QName { return f.tags }
