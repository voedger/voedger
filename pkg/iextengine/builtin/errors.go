/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginebuiltin

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func undefinedExtension(app appdef.AppQName, ext string) error {
	return fmt.Errorf("app %v: undefined extension: %v", app, ext)
}
