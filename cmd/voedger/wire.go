//go:generate go run github.com/google/wire/cmd/wire
//go:build wireinject
// +build wireinject

/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package main

import (
	"github.com/google/wire"

	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
	"github.com/voedger/voedger/pkg/ihttpimpl"
)

func wireServer(ihttp.CLIParams, ihttp.GrafanaPort, ihttp.PrometheusPort) (WiredServer, func(), error) {
	panic(
		wire.Build(
			ihttpimpl.NewProcessor,
			ihttpimpl.NewAPI,
			ihttpctl.NewHTTPProcessorController,
			apps.NewStaticEmbeddedResources,
			apps.NewRedirectionRoutes,
			apps.NewDefaultRedirectionRoute,
			wire.Struct(new(WiredServer), "*"),
		),
	)
}
