//go:generate go run github.com/google/wire/cmd/wire
//go:build wireinject
// +build wireinject

/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package main

import (
	"github.com/google/wire"

	voedger "github.com/voedger/voedger/cmd/voedger/voedgerimpl"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
	"github.com/voedger/voedger/pkg/ihttpimpl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/provider"
)

func wireServer(httpCliParams ihttp.CLIParams, appsCliParams voedger.CLIParams) (WiredServer, func(), error) {
	panic(
		wire.Build(
			ihttpimpl.NewProcessor,
			ihttpctl.NewHTTPProcessorController,
			ihttp.NewIRouterStorage,
			voedger.NewStaticEmbeddedResources,
			voedger.NewRedirectionRoutes,
			voedger.NewDefaultRedirectionRoute,
			voedger.NewAppStorageFactory,
			voedger.NewAppRequestHandlers,
			provideAppStorageProvider,
			wire.FieldsOf(&httpCliParams, "AcmeDomains"),
			wire.Struct(new(WiredServer), "*"),
		),
	)
}

// provideAppStorageProvider is intended to be used by wire instead of istorage/provider.Provide, because wire can not handle variadic arguments
func provideAppStorageProvider(appStorageFactory istorage.IAppStorageFactory) istorage.IAppStorageProvider {
	return provider.Provide(appStorageFactory)
}
