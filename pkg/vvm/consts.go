/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"time"

	istoragecas "github.com/heeus/core-istoragecas"
	router "github.com/untillpro/airs-router2"
)

const (
	NumCommandProcessors                 = 10
	NumQueryProcessors                   = 10 // <=0 -> 1 query processor will exist anyway
	DefaultQuotasChannels                = 1000
	DefaultQuotasChannelsPerSubject      = 10
	DefaultQuotasSubscriptions           = 10000
	DefaultQuotasSubscriptionsPerSubject = 20
	DefaultMetricsServicePort            = 8000
	DefaultPartitionsCount               = NumCommandProcessors // не кофигурировать. Временно столько же, сколько и командных процессоров
	DefaultCacheSize                     = 1024 * 1024 * 1024   // 1Gb
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

const (
	EPWSTemplates       EPKey = "WSTemplates"
	EPJournalIndices    EPKey = "JournalIndices"
	EPJournalPredicates EPKey = "JournalPredicates"
	EPPostDocs          EPKey = "PostDocs"
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
