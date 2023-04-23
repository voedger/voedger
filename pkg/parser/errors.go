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

func ErrSchemaContainsDuplicateName(schema, name string, pos *lexer.Position) error {
	return fmt.Errorf("schema %s contains duplicated name %s at pos %s", schema, name, pos.String())
}
