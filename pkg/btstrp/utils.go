/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package btstrp

import (
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

// SplitBuiltInApps extracts the cluster app and the remaining built-in apps,
// excluding apps deployed as sidecars. The remaining apps keep their input order.
func SplitBuiltInApps(builtinApps []appparts.BuiltInApp, sidecarApps []appparts.SidecarApp) (ClusterBuiltInApp, []appparts.BuiltInApp, error) {
	var clusterApp ClusterBuiltInApp
	otherApps := make([]appparts.BuiltInApp, 0, len(builtinApps))
	for _, app := range builtinApps {
		if app.Name == istructs.AppQName_sys_cluster {
			clusterApp = ClusterBuiltInApp(app)
		} else if !slices.ContainsFunc(sidecarApps, func(sa appparts.SidecarApp) bool {
			return sa.Name == app.Name
		}) {
			otherApps = append(otherApps, app)
		}
	}
	if clusterApp.Name == appdef.NullAppQName {
		return clusterApp, nil, fmt.Errorf("%s app should be added to VVM builtin apps", istructs.AppQName_sys_cluster)
	}
	return clusterApp, otherApps, nil
}
