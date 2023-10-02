/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registryapp

import "embed"

//go:embed schema.sql
var registrySchemaFS embed.FS
