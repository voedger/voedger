/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"os"

	ibus "github.com/untillpro/airs-ibus"
	router "github.com/untillpro/airs-router2"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokensjwt"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
)

func NewHVMDefaultConfig() HVMConfig {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	hvmCfg := HVMConfig{
		Routes:                 map[string]string{},
		RoutesRewrite:          map[string]string{},
		RouteDomains:           map[string]string{},
		RouterWriteTimeout:     router.DefaultRouterWriteTimeout, // same
		RouterReadTimeout:      router.DefaultRouterWriteTimeout, // same
		RouterConnectionsLimit: router.DefaultRouterConnectionsLimit,
		BLOBMaxSize:            DefaultBLOBMaxSize,
		TimeFunc:               DefaultTimeFunc,
		Name:                   commandprocessor.HVMName(hostname),
		HVMAppsBuilder:         HVMAppsBuilder{},
		BusTimeout:             BusTimeout(ibus.DefaultTimeout),
		BlobberServiceChannels: router.BlobberServiceChannels{
			{
				NumChannels:       1,
				ChannelBufferSize: 0,
			},
		},
		Quotas: in10n.Quotas{
			Channels:               DefaultQuotasChannels,
			ChannelsPerSubject:     DefaultQuotasChannelsPerSubject,
			Subsciptions:           DefaultQuotasSubscriptions,
			SubsciptionsPerSubject: DefaultQuotasSubscriptionsPerSubject,
		},
		PartitionsCount:      DefaultPartitionsCount,
		NumCommandProcessors: NumCommandProcessors,
		NumQueryProcessors:   NumQueryProcessors,
		StorageCacheSize:     DefaultCacheSize,
		MaxPrepareQueries:    DefaultMaxPrepareQueries,
		HVMPort:              DefaultHVMPort,
		MetricsServicePort:   DefaultMetricsServicePort,
		StorageFactory: func() (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragemem")
			return istorage.ProvideMem(), nil
		},
	}

	hvmCfg.AddProcessorChannel(
		// command processors
		// конкретный ресторан должен пойти в один и тотже cmd proc
		iprocbusmem.ChannelGroup{
			NumChannels:       NumCommandProcessors,
			ChannelBufferSize: NumCommandProcessors,
		},
		ProcessorChannel_Command,
	)

	hvmCfg.AddProcessorChannel(
		// query processors
		// все QueryProcessors сидят на одном канале, т.к. любой ресторан может обслуживаться любым query proc
		iprocbusmem.ChannelGroup{
			NumChannels:       1,
			ChannelBufferSize: 0,
		},
		ProcessorChannel_Query,
	)

	return hvmCfg
}

func (tsr *testISecretReader) ReadSecret(name string) ([]byte, error) {
	if name == SecretKeyJWTName {
		return itokensjwt.SecretKeyExample, nil
	}
	return tsr.realSecretReader.ReadSecret(name)
}

func (cfg *HVMConfig) AddProcessorChannel(cg iprocbusmem.ChannelGroup, t ProcessorChannelType) {
	cfg.processorsChannels = append(cfg.processorsChannels, ProcesorChannel{
		ChannelGroup: cg,
		ChannelType:  t,
	})
}

func (vvm *HVMConfig) ProvideServiceChannelFactory(procbus iprocbus.IProcBus) ServiceChannelFactory {
	return func(pct ProcessorChannelType, channelIdx int) iprocbus.ServiceChannel {
		for groupIdx, pcg := range vvm.processorsChannels {
			if pcg.ChannelType == pct {
				return procbus.ServiceChannel(groupIdx, channelIdx)
			}
		}
		panic("processor channel group type not found")
	}
}

func (ha *HVMApps) Exists(name istructs.AppQName) bool {
	for _, appQName := range *ha {
		if appQName == name {
			return true
		}
	}
	return false
}
