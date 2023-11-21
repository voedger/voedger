/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package apps

import (
	sysmonitor "github.com/voedger/voedger/pkg/apps/sys.monitor"
	"github.com/voedger/voedger/pkg/ihttpctl"
)

func NewStaticEmbeddedResources() []ihttpctl.StaticResourcesType {
	return []ihttpctl.StaticResourcesType{
		sysmonitor.New(),
	}
}

func NewRedirectionRoutes() ihttpctl.RedirectRoutes {
	return ihttpctl.RedirectRoutes{
		"(https?://[^/]*/)grafana/(.*)":    "http://127.0.0.1:3000/$2",
		"(https?://[^/]*/)prometheus/(.*)": "http://127.0.0.1:9090/$2",
	}
}

func NewDefaultRedirectionRoute() ihttpctl.DefaultRedirectRoute {
	return nil
}
