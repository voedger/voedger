/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

import "github.com/voedger/voedger/pkg/istorage"

type (
	// Version key
	VersionKey uint16

	// Version values
	VersionValue uint16
)

// Versions of system views
type Versions struct {
	storage istorage.IAppStorage
	vers    map[VersionKey]VersionValue
}
