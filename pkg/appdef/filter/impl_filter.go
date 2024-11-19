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

func (filter) And() func(func(appdef.IFilter) bool) { return func(func(appdef.IFilter) bool) {} }

func (filter) Not() appdef.IFilter { return nil }

func (filter) Or() func(func(appdef.IFilter) bool) { return func(func(appdef.IFilter) bool) {} }

func (filter) QNames() func(func(appdef.QName) bool) { return func(func(appdef.QName) bool) {} }

func (filter) Tags() func(func(string) bool) { return func(func(string) bool) {} }

func (filter) Types() func(func(appdef.TypeKind) bool) { return func(func(appdef.TypeKind) bool) {} }

// allMatches returns types that match the filter.
func allMatches(f appdef.IFilter, types appdef.SeqType) appdef.SeqType {
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
