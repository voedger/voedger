/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"os"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/processors"

	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/router"
)

func NewVVMDefaultConfig() VVMConfig {
	hostname, err := os.Hostname()
	if err != nil {
		// notest
		panic(err)
	}
	res := VVMConfig{
		Routes:                 map[string]string{},
		RoutesRewrite:          map[string]string{},
		RouteDomains:           map[string]string{},
		RouterWriteTimeout:     router.DefaultRouterWriteTimeout, // same
		RouterReadTimeout:      router.DefaultRouterWriteTimeout, // same
		RouterConnectionsLimit: router.DefaultConnectionsLimit,
		BLOBMaxSize:            DefaultBLOBMaxSize,
		Time:                   coreutils.NewITime(),
		Name:                   processors.VVMName(hostname),
		VVMAppsBuilder:         VVMAppsBuilder{},
		SendTimeout:            bus.DefaultSendTimeout,
		NumCommandProcessors:   DefaultNumCommandProcessors,
		NumQueryProcessors:     DefaultNumQueryProcessors,
		NumBLOBProcessors:      DefaultNumBLOBProcessors,
		StorageCacheSize:       DefaultCacheSize,
		MaxPrepareQueries:      DefaultMaxPrepareQueries,
		VVMPort:                DefaultVVMPort,
		MetricsServicePort:     DefaultMetricsServicePort,
		StorageFactory: func() (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragemem")
			return mem.Provide(coreutils.MockTime), nil
		},
		SecretsReader: isecretsimpl.ProvideSecretReader(),
		IP:            coreutils.LocalhostIP,
		NumVVM:        1,

		// [~tuc.VVMConfig.ConfigureSequencesTrustLevel~]
		SequencesTrustLevel: isequencer.SequencesTrustLevel_0,
	}
	if coreutils.IsTest() {
		res.SecretsReader = itokensjwt.ProvideTestSecretsReader(res.SecretsReader)
	}
	return res
}

func (cfg *VVMConfig) addProcessorChannel(cg iprocbusmem.ChannelGroup, t ProcessorChannelType) {
	cfg.processorsChannels = append(cfg.processorsChannels, ProcesorChannel{
		ChannelGroup: cg,
		ChannelType:  t,
	})
}

func (cfg *VVMConfig) ProvideServiceChannelFactory(procbus iprocbus.IProcBus) ServiceChannelFactory {
	return func(pct ProcessorChannelType, channelIdx uint) iprocbus.ServiceChannel {
		for groupIdx, pcg := range cfg.processorsChannels {
			if pcg.ChannelType == pct {
				return procbus.ServiceChannel(uint(groupIdx), channelIdx) // nolint G115
			}
		}
		panic("processor channel group type not found")
	}
}

func (ha *VVMApps) Exists(name appdef.AppQName) bool {
	for _, appQName := range *ha {
		if appQName == name {
			return true
		}
	}
	return false
}
