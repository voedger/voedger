/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

var ErrDirContainsNoSchemaFiles = errors.New("directory contains no schema files")
var ErrDirContainsDifferentSchemas = errors.New("directory contains files for different schemas")
var ErrFunctionParamsIncorrect = errors.New("function parameters do not match")
var ErrFunctionResultIncorrect = errors.New("function result do not match")
var ErrFunctionNotFound = errors.New("function not found")

func ErrSchemaContainsDuplicateName(schema, name string) error {
	return fmt.Errorf("schema '%s' contains duplicated name %s", schema, name)
}

func errorAt(err error, pos *lexer.Position) error {
	return fmt.Errorf("%s: %w", pos.String(), err)
}
