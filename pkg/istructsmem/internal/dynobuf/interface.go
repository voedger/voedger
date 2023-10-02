/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
)

// Dynobuffer schemes.
//
// Pass appdef.IAppDef to Prepare() method to prepare schemes.
//
// Use Scheme() method to get scheme for any structured type (doc or record)
// or for view value scheme.
//
// Use ViewPartKeyScheme() method to get view partition key scheme
// and ViewClustColsScheme() method to get view clustering columns scheme.
type DynoBufSchemes struct {
	schemes map[string]*dynobuffers.Scheme
}
