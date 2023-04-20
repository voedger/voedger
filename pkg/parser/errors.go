/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"errors"
	"fmt"
)

var ErrDirContainsNoSchemaFiles = errors.New("directory contains no schema files")
var ErrDirContainsDifferentSchemas = errors.New("directory contains files for different schemas")

func ErrSchemaContainsDuplicateName(schema, name string) error {
	return fmt.Errorf("schema %s contains duplicated name %s", schema, name)
}
