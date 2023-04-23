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

func ErrFunctionNotFound(name OptQName, pos *lexer.Position) error {
	if name.Package == "" {
		return fmt.Errorf("function %s not found at pos: %s", name.Name, pos.String())
	}
	return fmt.Errorf("function %s.%s not found at pos: %s", name.Package, name.Name, pos.String())
}

func ErrSchemaContainsDuplicateName(schema, name string, pos *lexer.Position) error {
	return fmt.Errorf("schema %s contains duplicated name %s at pos %s", schema, name, pos.String())
}
