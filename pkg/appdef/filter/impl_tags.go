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
}

func makeTagsFilter(tag string, tags ...string) appdef.IFilter {
	f := &tagsFilter{tags: make(map[string]bool)}
	f.tags[tag] = true
	for _, t := range tags {
		f.tags[t] = true
	}
	return f
}

func (tagsFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Tags }

func (f tagsFilter) Match(t appdef.IType) bool {
	// TODO: implement when appdef.IType.Tags() is implemented
	// return f.tags[t.Tag()]
	return true
}

func (f tagsFilter) String() string {
	return fmt.Sprintf("filter %s %v", f.Kind().TrimString(), f.Tags())
}

func (f tagsFilter) Tags() []string {
	return slices.Sorted(maps.Keys(f.tags))
}
