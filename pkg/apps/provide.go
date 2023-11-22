/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package apps

import (
	"fmt"

	sysmonitor "github.com/voedger/voedger/pkg/apps/sys.monitor"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
)

func NewStaticEmbeddedResources() []ihttpctl.StaticResourcesType {
	return []ihttpctl.StaticResourcesType{
		sysmonitor.New(),
	}
}

func NewRedirectionRoutes(grafanaPort ihttp.GrafanaPort, prometheusPort ihttp.PrometheusPort) ihttpctl.RedirectRoutes {
	return ihttpctl.RedirectRoutes{
		"(https?://[^/]*)/grafana($|/.*)":    fmt.Sprintf("http://127.0.0.1:%d$2", grafanaPort),
		"(https?://[^/]*)/prometheus($|/.*)": fmt.Sprintf("http://127.0.0.1:%d$2", prometheusPort),
	}
}

func NewDefaultRedirectionRoute() ihttpctl.DefaultRedirectRoute {
	return nil
}
