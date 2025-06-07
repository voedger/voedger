/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"time"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/router"
)

const (
	DefaultNumCommandProcessors          istructs.NumCommandProcessors = 10
	DefaultNumQueryProcessors            istructs.NumQueryProcessors   = 10   // <=0 -> 1 query processor will exist anyway
	DefaultNumBLOBProcessors             istructs.NumBLOBProcessors    = 10   // <=0 -> 1 BLOB processor will exist anyway
	DefaultQuotasChannelsFactor                                        = 1000 // Quotas.Channels will be NumCommandProcessors * DefaultQuotasChannelsFactor
	DefaultQuotasChannelsPerSubject                                    = 50
	DefaultQuotasSubscriptionsFactor                                   = 1000 // Quotas.Subscriptions will be NumCommandProcessors * DefaultQuotasSubscriptionsFactor
	DefaultQuotasSubscriptionsPerSubject                               = 100
	DefaultMetricsServicePort                                          = 8000
	DefaultCacheSize                                                   = 1024 * 1024 * 1024 // 1Gb
	ShortestPossibleFunctionNameLen                                    = len("q.a.a")
	DefaultMaxPrepareQueries                                           = 10
	DefaultBLOBMaxSize                                                 = iblobstorage.BLOBMaxSizeType(20971520) // 20Mb
	DefaultVVMPort                                                     = router.DefaultPort
	actualizerFlushInterval                                            = time.Millisecond * 500
	DefaultLeadershipDurationSeconds                                   = ielections.LeadershipDurationSeconds(20)
	DefaultLeadershipAcquisitionDuration                               = LeadershipAcquisitionDuration(120 * time.Second)
)

const (
	ProcessorChannel_Command ProcessorChannelType = iota
	ProcessorChannel_Query_V1
	ProcessorChannel_BLOB
	ProcessorChannel_Query_V2
)

var (
	LocalHost = "http://127.0.0.1"
)
