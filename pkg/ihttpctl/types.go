/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package ihttpctl

import "io/fs"

type StaticResourcesType map[string]fs.FS
type RedirectRoutes map[string]string
type DefaultRedirectRoute map[string]string // single record only
type AcmeDomains []string
