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

// tagsFilter is a filter that matches types by their tags.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type tagsFilter struct {
	filter
	tags appdef.QNames
}

func makeTagsFilter(tag appdef.QName, tags ...appdef.QName) appdef.IFilter {
	f := &tagsFilter{tags: appdef.QNamesFrom(tag)}
	for _, t := range tags {
		f.tags.Add(t)
	}
	return f
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
	s := fmt.Sprintf("filter.%s(", f.Kind().TrimString())
	for i, c := range f.tags {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}

func (f tagsFilter) Tags() func(func(appdef.QName) bool) {
	return slices.Values(f.tags)
}
