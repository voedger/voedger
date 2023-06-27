/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"time"

	router "github.com/untillpro/airs-router2"
	"github.com/voedger/voedger/pkg/istorageimpl/istoragecas"
)

const (
	DefaultNumCommandProcessors          = 10
	DefaultNumQueryProcessors            = 10  // <=0 -> 1 query processor will exist anyway
	DefaultQuotasChannelsFactor          = 100 // Quotas.Channels will be NumCommandProcessors * DefaultQuotasFactor
	DefaultQuotasChannelsPerSubject      = 10
	DefaultQuotasSubscriptionsFactor     = 1000 // Quotas.Subscriptions will be NumCommandProcessors * DefaultQuotasSubscriptionsFactor
	DefaultQuotasSubscriptionsPerSubject = 20
	DefaultMetricsServicePort            = 8000
	DefaultCacheSize                     = 1024 * 1024 * 1024 // 1Gb
	ShortestPossibleFunctionNameLen      = len("q.a.a")
	DefaultBLOBWorkersNum                = 10
	DefaultRetryAfterSecondsOn503        = 1
	DefaultMaxPrepareQueries             = 10
	DefaultBLOBMaxSize                   = router.BLOBMaxSizeType(20971520) // 20Mb
	DefaultVVMPort                       = router.DefaultRouterPort
	SecretKeyJWTName                     = "secretKeyJWT"
	actualizerIntentsLimit               = 128
)

const (
	ProcessorChannel_Command ProcessorChannelType = iota
	ProcessorChannel_Query
)

var (
	LocalHost        = "http://127.0.0.1"
	DefaultTimeFunc  = time.Now
	DefaultCasParams = istoragecas.CassandraParamsType{
		Hosts:                   "127.0.0.1",
		Port:                    9042,
		KeyspaceWithReplication: istoragecas.SimpleWithReplication,
	}
)
