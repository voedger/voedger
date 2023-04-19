/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import "errors"

var ErrDirContainsNoSchemaFiles = errors.New("directory contains no schema files")
var ErrDirContainsDifferentSchemas = errors.New("directory contains files for different schemas")
