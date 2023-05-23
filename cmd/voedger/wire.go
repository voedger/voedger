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
	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ibusmem"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
	"github.com/voedger/voedger/pkg/ihttpimpl"
)

func wireServer(ibus.CLIParams, ihttp.CLIParams) (WiredServer, func(), error) {
	panic(
		wire.Build(
			ibusmem.New,
			ihttpimpl.NewProcessor,
			ihttpimpl.NewAPI,
			ihttpctl.NewHTTPProcessorController,
			apps.ProvideStaticEmbeddedResources,
			wire.Struct(new(WiredServer), "*"),
		),
	)
}
