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
		"(https?://[^/]*)/grafana($|/.*)":    "http://127.0.0.1:3000/$3",
		"(https?://[^/]*)/prometheus($|/.*)": "http://127.0.0.1:9090/$3",
	}
}

func NewDefaultRedirectionRoute() ihttpctl.DefaultRedirectRoute {
	return nil
}
