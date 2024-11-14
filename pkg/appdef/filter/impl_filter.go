/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

type filter struct{}

func (filter) And() []IFilter { return nil }

func (filter) Kind() appdef.FilterKind { return appdef.FilterKind_null }

func (filter) Not() IFilter { return nil }

func (filter) Or() []IFilter { return nil }

func (filter) QNames() appdef.QNames { return nil }

func (filter) Match(appdef.IType) bool { return false }

func (filter) Matches(appdef.IWithTypes) appdef.QNames { return nil }

func (filter) Tags() []string { return nil }

func (filter) Types() appdef.TypeKindSet { return appdef.TypeKindSet{} }
