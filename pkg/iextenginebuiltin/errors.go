/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginebuiltin

import "errors"

func undefinedExtension(name string) error {
	return errors.New("undefined extension: " + name)
}
