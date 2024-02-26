/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewazero

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

func undefinedPackage(name string) error {
	return errors.New("undefined package: " + name)
}

func incorrectExtensionName(name string) error {
	return errors.New("incorrect extension name: " + name)
}

func invalidExtensionName(name string) error {
	return errors.New("invalid extension name: " + name)
}
