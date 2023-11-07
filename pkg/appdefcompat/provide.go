/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func CheckBackwardCompatibility(old, new appdef.IAppDef) (cerrs *CompatibilityErrors) {
	return checkBackwardCompatibility(old, new)
}

func IgnoreCompatibilityErrors(cerrs *CompatibilityErrors, pathsToIgnore [][]string) (cerrsOut *CompatibilityErrors) {
	return ignoreCompatibilityErrors(cerrs, pathsToIgnore)
}
