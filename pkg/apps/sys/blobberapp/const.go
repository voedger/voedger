/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package blobberapp

import "embed"

//go:embed schema.sql
var blobberSchemaFS embed.FS

const BlobberAppFQN = "github.com/voedger/voedger/pkg/apps/sys/blobberapp"
