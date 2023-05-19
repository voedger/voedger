/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

const UnknownVersion VersionValue = 0

const (
	// version key for QNames system view
	SysQNamesVersion VersionKey = iota + 1

	// version key for containers names system view
	SysContainersVersion

	// version key for singletons system view
	SysSingletonsVersion

	// version key for uniques system view
	SysUniquesVersion
)
