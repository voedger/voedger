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

// typesFilter is a filter that matches types by their kind.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type typesFilter struct {
	filter
	types   []appdef.TypeKind
	typeSet appdef.TypeKindSet
}

func makeTypesFilter(tt ...appdef.TypeKind) typesFilter {
	if len(tt) == 0 {
		panic("types filter should have at least one type")
	}
	f := typesFilter{typeSet: set.From(tt...)}
	f.types = f.typeSet.AsArray()
	return f
}

func newTypesFilter(tt ...appdef.TypeKind) *typesFilter {
	f := makeTypesFilter(tt...)
	return &f
}

func (typesFilter) Kind() appdef.FilterKind { return appdef.FilterKind_Types }

func (f typesFilter) Match(t appdef.IType) bool {
	return f.typeSet.Contains(t.Kind())
}

func (f typesFilter) String() string {
	var s string
	if t, ok := typesStringDecorators[string(f.typeSet.AsBytes())]; ok {
		s = t
	} else {
		// TYPES(…) FROM …)
		s = "TYPES("
		for i, t := range f.types {
			if i > 0 {
				s += ", "
			}
			s += t.TrimString()
		}
		s += ")"
	}
	return s
}

func (f typesFilter) Types() []appdef.TypeKind { return f.types }

var typesStringDecorators = map[string]string{
	string(appdef.TypeKind_Structures.AsBytes()): "ALL TABLES",
	string(appdef.TypeKind_Functions.AsBytes()):  "ALL FUNCTIONS",
}

// wsTypesFilter is a filter that matches types by their kind.
// Matched types should be located in the specified workspace.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type wsTypesFilter struct {
	typesFilter
	ws appdef.QName
}

func newWSTypesFilter(ws appdef.QName, tt ...appdef.TypeKind) *wsTypesFilter {
	if ws == appdef.NullQName {
		panic("workspace should be specified")
	}
	return &wsTypesFilter{
		typesFilter: makeTypesFilter(tt...),
		ws:          ws,
	}
}

func (f wsTypesFilter) Match(t appdef.IType) bool {
	if f.typesFilter.Match(t) {
		ws := t.Workspace()
		return (ws != nil) && (ws.QName() == f.ws)
	}
	return false
}

func (f wsTypesFilter) String() string {
	return fmt.Sprintf("%s FROM %s", f.typesFilter.String(), f.ws)
}

func (f wsTypesFilter) WS() appdef.QName { return f.ws }
