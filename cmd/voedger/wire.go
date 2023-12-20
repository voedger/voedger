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
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
)

func wireServer(ihttp.CLIParams, apps.CLIParams, ihttp.GrafanaPort, ihttp.PrometheusPort) (WiredServer, func(), error) {
	panic(
		wire.Build(
			ihttpimpl.NewProcessor,
			ihttpimpl.NewAPI,
			ihttpctl.NewHTTPProcessorController,
			ihttp.NewIRouterStorage,
			apps.NewStaticEmbeddedResources,
			apps.NewRedirectionRoutes,
			apps.NewDefaultRedirectionRoute,
			apps.NewAppStorageFactory,
			provideAppStorageProvider,
			emptyAcmeDomainList,
			wire.Struct(new(WiredServer), "*"),
		),
	)
}

func provideAppStorageProvider(appStorageFactory istorage.IAppStorageFactory) istorage.IAppStorageProvider {
	return istorageimpl.Provide(appStorageFactory)
}

func emptyAcmeDomainList() ihttpctl.AcmeDomains {
	return ihttpctl.AcmeDomains{}
}
