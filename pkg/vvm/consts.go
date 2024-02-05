/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"time"

	istoragecas2 "github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/router"
)

const (
	DefaultNumCommandProcessors          = 10
	DefaultNumQueryProcessors            = 10  // <=0 -> 1 query processor will exist anyway
	DefaultQuotasChannelsFactor          = 100 // Quotas.Channels will be NumCommandProcessors * DefaultQuotasFactor
	DefaultQuotasChannelsPerSubject      = 50
	DefaultQuotasSubscriptionsFactor     = 1000 // Quotas.Subscriptions will be NumCommandProcessors * DefaultQuotasSubscriptionsFactor
	DefaultQuotasSubscriptionsPerSubject = 100
	DefaultMetricsServicePort            = 8000
	DefaultCacheSize                     = 1024 * 1024 * 1024 // 1Gb
	ShortestPossibleFunctionNameLen      = len("q.a.a")
	DefaultBLOBWorkersNum                = 10
	DefaultRetryAfterSecondsOn503        = 1
	DefaultMaxPrepareQueries             = 10
	DefaultBLOBMaxSize                   = router.BLOBMaxSizeType(20971520) // 20Mb
	DefaultVVMPort                       = router.DefaultRouterPort
	actualizerFlushInterval              = time.Millisecond * 500
	defaultCassandraPort                 = 9042
)

const (
	ProcessorChannel_Command ProcessorChannelType = iota
	ProcessorChannel_Query
)

var (
	LocalHost        = "http://127.0.0.1"
	DefaultTimeFunc  = time.Now
	DefaultCasParams = istoragecas2.CassandraParamsType{
		Hosts:                   "127.0.0.1",
		Port:                    defaultCassandraPort,
		KeyspaceWithReplication: istoragecas2.SimpleWithReplication,
	}
)
