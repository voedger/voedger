/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func CheckBackwardCompatibility(oldAppDef, newAppDef appdef.IAppDef) (cerrs *CompatibilityErrors) {
	return checkBackwardCompatibility(oldAppDef, newAppDef)
}

func IgnoreCompatibilityErrors(cerrs *CompatibilityErrors, pathsToIgnore [][]string) (cerrsOut *CompatibilityErrors) {
	return ignoreCompatibilityErrors(cerrs, pathsToIgnore)
}
