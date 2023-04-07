//go:generate go run github.com/google/wire/cmd/wire
//go:build wireinject
// +build wireinject

/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package ce

import (
	"github.com/google/wire"
	"github.com/untillpro/voedger/pkg/apps"
	"github.com/untillpro/voedger/pkg/ibus"
	"github.com/untillpro/voedger/pkg/ibusmem"
	"github.com/untillpro/voedger/pkg/ihttp"
	"github.com/untillpro/voedger/pkg/ihttpctl"
	"github.com/untillpro/voedger/pkg/ihttpimpl"
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
