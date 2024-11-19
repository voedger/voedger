/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// abstract filter.
type filter struct{}

func (filter) And() []appdef.IFilter { return nil }

func (filter) Not() appdef.IFilter { return nil }

func (filter) Or() []appdef.IFilter { return nil }

func (filter) QNames() func(func(appdef.QName) bool) { return func(func(appdef.QName) bool) {} }

func (filter) Tags() []string { return nil }

func (filter) Types() appdef.TypeKindSet { return appdef.TypeKindSet{} }

// allMatches returns types that match the filter.
func allMatches(f appdef.IFilter, types appdef.IterTypes) appdef.IterTypes {
	return func(visit func(appdef.IType) bool) {
		for t := range types {
			if f.Match(t) {
				if !visit(t) {
					return
				}
			}
		}
	}
}
