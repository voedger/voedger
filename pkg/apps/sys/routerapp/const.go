/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package routerapp

import "embed"

//go:embed schema.sql
var routerSchemaFS embed.FS

const RouterAppFQN = "github.com/voedger/voedger/pkg/apps/sys/routerapp"
