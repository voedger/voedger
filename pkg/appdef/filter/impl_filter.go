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

// trueFilter realizes filter what always matches any type.
//
// # Supports:
//   - appdef.IFilter.
//   - fmt.Stringer
type trueFilter struct{ filter }

func (trueFilter) Kind() appdef.FilterKind   { return appdef.FilterKind_True }
func (trueFilter) Match(t appdef.IType) bool { return true }
func (trueFilter) String() string            { return "TRUE" }

var trueFlt *trueFilter = &trueFilter{}
