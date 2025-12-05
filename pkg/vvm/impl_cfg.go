/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"os"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/sys/storages"

	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
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
		Time:                   timeu.NewITime(),
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
		StorageFactory: func(time timeu.ITime) (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragemem")
			return mem.Provide(time), nil
		},
		SecretsReader:                    isecretsimpl.ProvideSecretReader(),
		IP:                               httpu.LocalhostIP,
		NumVVM:                           1,
		AdminPort:                        DefaultAdminPort,
		EmailSender:                      storages.NewIEmailSenderSMTP(),
		SchemasCache:                     &NullSchemasCache{},
		PolicyOptsForFederationWithRetry: httpu.DefaultRetryPolicyOpts,

		// [~server.design.sequences/tuc.VVMConfig.ConfigureSequencesTrustLevel~impl]
		SequencesTrustLevel: isequencer.SequencesTrustLevel_0,
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
