/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"os"

	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokensjwt"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
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
		RouterConnectionsLimit: router.DefaultRouterConnectionsLimit,
		BLOBMaxSize:            DefaultBLOBMaxSize,
		TimeFunc:               DefaultTimeFunc,
		Name:                   commandprocessor.VVMName(hostname),
		VVMAppsBuilder:         VVMAppsBuilder{},
		BusTimeout:             BusTimeout(ibus.DefaultTimeout),
		BlobberServiceChannels: router.BlobberServiceChannels{
			{
				NumChannels:       1,
				ChannelBufferSize: 0,
			},
		},
		NumCommandProcessors: DefaultNumCommandProcessors,
		NumQueryProcessors:   DefaultNumQueryProcessors,
		StorageCacheSize:     DefaultCacheSize,
		MaxPrepareQueries:    DefaultMaxPrepareQueries,
		VVMPort:              DefaultVVMPort,
		MetricsServicePort:   DefaultMetricsServicePort,
		StorageFactory: func() (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragemem")
			return istorage.ProvideMem(), nil
		},
	}
	return res
}

func (tsr *testISecretReader) ReadSecret(name string) ([]byte, error) {
	if name == SecretKeyJWTName {
		return itokensjwt.SecretKeyExample, nil
	}
	return tsr.realSecretReader.ReadSecret(name)
}

func (cfg *VVMConfig) addProcessorChannel(cg iprocbusmem.ChannelGroup, t ProcessorChannelType) {
	cfg.processorsChannels = append(cfg.processorsChannels, ProcesorChannel{
		ChannelGroup: cg,
		ChannelType:  t,
	})
}

func (cfg *VVMConfig) ProvideServiceChannelFactory(procbus iprocbus.IProcBus) ServiceChannelFactory {
	return func(pct ProcessorChannelType, channelIdx int) iprocbus.ServiceChannel {
		for groupIdx, pcg := range cfg.processorsChannels {
			if pcg.ChannelType == pct {
				return procbus.ServiceChannel(groupIdx, channelIdx)
			}
		}
		panic("processor channel group type not found")
	}
}

func (ha *VVMApps) Exists(name istructs.AppQName) bool {
	for _, appQName := range *ha {
		if appQName == name {
			return true
		}
	}
	return false
}
