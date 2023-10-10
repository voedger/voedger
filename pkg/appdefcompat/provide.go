/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func CheckBackwardCompatibility(oldBuilder, newBuilder appdef.IAppDefBuilder) (cerrs *CompatibilityErrors, err error) {
	return checkBackwardCompatibility(oldBuilder, newBuilder)
}

func IgnoreCompatibilityErrors(cerrs *CompatibilityErrors, toBeIgnored []CompatibilityError) (cerrsOut *CompatibilityErrors) {
	return ignoreCompatibilityErrors(cerrs, toBeIgnored)
}
