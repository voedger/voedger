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

func (filter) And() []appdef.IFilter    { return nil }
func (filter) Not() appdef.IFilter      { return nil }
func (filter) Or() []appdef.IFilter     { return nil }
func (filter) QNames() []appdef.QName   { return nil }
func (filter) Tags() []appdef.QName     { return nil }
func (filter) Types() []appdef.TypeKind { return nil }
func (filter) WS() appdef.QName         { return appdef.NullQName }

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
