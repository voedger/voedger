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

func wireServer(httpCliParams ihttp.CLIParams, appsCliParams apps.CLIParams) (WiredServer, func(), error) {
	panic(
		wire.Build(
			ihttpimpl.NewProcessor,
			ihttpctl.NewHTTPProcessorController,
			ihttp.NewIRouterStorage,
			apps.NewStaticEmbeddedResources,
			apps.NewRedirectionRoutes,
			apps.NewDefaultRedirectionRoute,
			apps.NewAppStorageFactory,
			provideAppStorageProvider,
			wire.FieldsOf(&httpCliParams, "AcmeDomains"),
			wire.Struct(new(WiredServer), "*"),
		),
	)
}

// provideAppStorageProvider is intended to be used by wire instead of istorageimpl.Provide, because wire can not handle variadic arguments
func provideAppStorageProvider(appStorageFactory istorage.IAppStorageFactory) istorage.IAppStorageProvider {
	return istorageimpl.Provide(appStorageFactory)
}
