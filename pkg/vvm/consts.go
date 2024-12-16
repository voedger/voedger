/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"time"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/router"
)

const (
	DefaultNumCommandProcessors          istructs.NumCommandProcessors = 10
	DefaultNumQueryProcessors            istructs.NumQueryProcessors   = 10  // <=0 -> 1 query processor will exist anyway
	DefaultQuotasChannelsFactor                                        = 1000 // Quotas.Channels will be NumCommandProcessors * DefaultQuotasChannelsFactor
	DefaultQuotasChannelsPerSubject                                    = 50
	DefaultQuotasSubscriptionsFactor                                   = 1000 // Quotas.Subscriptions will be NumCommandProcessors * DefaultQuotasSubscriptionsFactor
	DefaultQuotasSubscriptionsPerSubject                               = 100
	DefaultMetricsServicePort                                          = 8000
	DefaultCacheSize                                                   = 1024 * 1024 * 1024 // 1Gb
	ShortestPossibleFunctionNameLen                                    = len("q.a.a")
	DefaultBLOBWorkersNum                                              = 10
	DefaultRetryAfterSecondsOn503                                      = 1
	DefaultMaxPrepareQueries                                           = 10
	DefaultBLOBMaxSize                                                 = iblobstorage.BLOBMaxSizeType(20971520) // 20Mb
	DefaultVVMPort                                                     = router.DefaultPort
	actualizerFlushInterval                                            = time.Millisecond * 500
	defaultCassandraPort                                               = 9042
)

const (
	ProcessorChannel_Command ProcessorChannelType = iota
	ProcessorChannel_Query
)

var (
	LocalHost        = "http://127.0.0.1"
	DefaultCasParams = cas.CassandraParamsType{
		Hosts:                   "127.0.0.1",
		Port:                    defaultCassandraPort,
		KeyspaceWithReplication: cas.SimpleWithReplication,
	}
)
