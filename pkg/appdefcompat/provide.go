/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func CheckBackwardCompatibility(old, new appdef.IAppDef) (cerrs CompatibilityErrors) {
	return checkBackwardCompatibility(old, new)
}

// cerrsOut: all cerrsIn that are not in toBeIgnored
// toBeIgnored.Pos is ignored in comparison
func IgnoreCompatibilityErrors(cerrs CompatibilityErrors, toBeIgnored []CompatibilityError) (cerrsOut CompatibilityErrors) {
	return
}
