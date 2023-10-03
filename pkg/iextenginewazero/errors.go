/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextenginewasm

import "errors"

var (
	ErrUnableToReadMemory = errors.New("unable to read result from WASM module")

	PanicIncorrectKey        = "incorrect key"
	PanicIncorrectKeyBuilder = "incorrect key builder"
	PanicIncorrectValue      = "incorrect value"
	PanicIncorrectIntent     = "incorrect intent"
)

func missingExportedFunction(name string) error {
	return errors.New("missing exported function: " + name)
}

func incorrectExtensionName(name string) error {
	return errors.New("incorrect extension name: " + name)
}

func invalidExtensionName(name string) error {
	return errors.New("invalid extension name: " + name)
}
