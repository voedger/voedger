/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"sort"

	"github.com/voedger/voedger/pkg/appdef"
)

type filter struct{}

func (filter) And() []appdef.IFilter { return nil }

func (filter) Kind() appdef.FilterKind { return appdef.FilterKind_null }

func (filter) Not() appdef.IFilter { return nil }

func (filter) Or() []appdef.IFilter { return nil }

func (filter) QNames() appdef.QNames { return nil }

func (filter) Match(appdef.IType) bool { return false }

func (filter) Matches(appdef.IWithTypes) appdef.IWithTypes { return NullTypes }

func (filter) Tags() []string { return nil }

func (filter) Types() appdef.TypeKindSet { return appdef.TypeKindSet{} }

// Used to collect filtered types. Supports appdef.IWithTypes
type types struct {
	m map[appdef.QName]appdef.IType
	s []appdef.IType
}

func makeTypes() *types {
	return &types{m: make(map[appdef.QName]appdef.IType)}
}

func cloneTypes(tt appdef.IWithTypes) *types {
	clone := makeTypes()
	for t := range tt.Types {
		clone.add(t)
	}
	return clone
}

func (tt types) String() string {
	s := ""
	for t := range tt.Types {
		if s != "" {
			s += ", "
		}
		s += fmt.Sprint(t)
	}
	return fmt.Sprintf("[%s]", s)
}

func (tt types) Type(name appdef.QName) appdef.IType {
	if t, ok := tt.m[name]; ok {
		return t
	}
	return appdef.NullType
}

func (tt types) TypeCount() int {
	return len(tt.m)
}

func (tt *types) Types(visit func(appdef.IType) bool) {
	if len(tt.s) != len(tt.m) {
		tt.s = make([]appdef.IType, 0, len(tt.m))
		for _, t := range tt.m {
			tt.s = append(tt.s, t)
		}
		sort.Slice(tt.s, func(i, j int) bool {
			return tt.s[i].QName().String() < tt.s[j].QName().String()
		})
	}
	for _, t := range tt.s {
		if !visit(t) {
			break
		}
	}
}

func (tt *types) add(t appdef.IType) {
	tt.m[t.QName()] = t
	tt.s = nil
}

func (tt *types) delete(t appdef.IType) {
	delete(tt.m, t.QName())
	tt.s = nil
}
