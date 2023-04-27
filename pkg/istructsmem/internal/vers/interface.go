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
//
//	Use Get() to read a version of system view.
//	Use Put() to write a version of system view.
//	Use Prepare() to load Versions from storage.
type Versions struct {
	storage istorage.IAppStorage
	vers    map[VersionKey]VersionValue
}
