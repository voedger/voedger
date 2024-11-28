/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"iter"

	"github.com/voedger/voedger/pkg/appdef"
)

// abstract filter.
type filter struct{}

func (filter) And() iter.Seq[appdef.IFilter] { return func(func(appdef.IFilter) bool) {} }

func (filter) Not() appdef.IFilter { return nil }

func (filter) Or() iter.Seq[appdef.IFilter] { return func(func(appdef.IFilter) bool) {} }

func (filter) QNames() iter.Seq[appdef.QName] { return func(func(appdef.QName) bool) {} }

func (filter) Tags() iter.Seq[appdef.QName] { return func(func(appdef.QName) bool) {} }

func (filter) Types() iter.Seq[appdef.TypeKind] { return func(func(appdef.TypeKind) bool) {} }

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
