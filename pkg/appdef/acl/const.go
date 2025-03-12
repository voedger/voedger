/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/set"
)

var publushedTypes = func() appdef.TypeKindSet {
	s := set.Collect(appdef.TypeKind_Structures.Values())
	s.Set(appdef.TypeKind_ViewRecord)
	s.Set(appdef.TypeKind_Functions.AsArray()...)
	s.SetReadOnly()
	return s
}()

var publishedOperations = func() appdef.OperationsSet {
	s := set.Collect(appdef.RecordsOperations.Values())
	s.Set(appdef.OperationKind_Execute)
	s.SetReadOnly()
	return s
}()
