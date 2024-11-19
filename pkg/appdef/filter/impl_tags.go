/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"maps"
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
	tags map[string]bool
	s    []string
}

func makeTagsFilter(tag string, tags ...string) appdef.IFilter {
	f := &tagsFilter{tags: make(map[string]bool)}
	f.tags[tag] = true
	for _, t := range tags {
		f.tags[t] = true
	}
	f.s = slices.Sorted(maps.Keys(f.tags))
	return f
}

func (tagsFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Tags }

func (f tagsFilter) Match(t appdef.IType) bool {
	// TODO: implement when appdef.IType.Tags() is implemented
	// return f.tags[t.Tag()]
	return true
}

func (f tagsFilter) String() string {
	s := fmt.Sprintf("filter.%s(", f.Kind().TrimString())
	for i, c := range f.s {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(c)
	}
	return s + ")"
}

func (f tagsFilter) Tags() func(func(string) bool) {
	return slices.Values(f.s)
}
