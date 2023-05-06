/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

// Loads all uniques IDs from storage, add all uniques from application definitions and store if some changes.
// Must be called at application starts
func PrepareApDefUniqueIDs(storage istorage.IAppStorage, versions *vers.Versions, qnames *qnames.QNames, appDef appdef.IAppDef) (err error) {
	return newUniques().prepare(storage, versions, qnames, appDef)
}
