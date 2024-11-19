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

func (filter) QNames() appdef.QNames { return nil }

func (filter) Tags() []string { return nil }

func (filter) Types() appdef.TypeKindSet { return appdef.TypeKindSet{} }
