/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"slices"
	"strings"

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
	return slices.ContainsFunc(f.tags, t.HasTag)
}

func (f tagsFilter) String() string {
	// TAGS(…)
	parts := make([]string, len(f.tags))
	for i, c := range f.tags {
		parts[i] = c.String()
	}
	return "TAGS(" + strings.Join(parts, ", ") + ")"
}

func (f tagsFilter) Tags() []appdef.QName { return f.tags }
