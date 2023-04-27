/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	istructs "github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
)

type TimeAfterFunc func(d time.Duration) <-chan time.Time

type LogErrorFunc func(args ...interface{})

type AsyncActualizerConf struct {
	Ctx          context.Context
	AppQName     istructs.AppQName
	AppStructs   AppStructsFunc
	SecretReader isecrets.ISecretReader
	Partition    istructs.PartitionID
	// Optional. Default value: `time.After`
	AfterError TimeAfterFunc
	// Optional. Default value: `core-logger.Error`
	LogError LogErrorFunc
	// Optional.
	Metrics AsyncActualizerMetrics
	//IntentsLimit top limit per event, optional, default value is 100
	IntentsLimit int
	//BundlesLimit top limit when bundle size is greater than this value, actualizer flushes changes to underlying storage, optional, default value is 100
	BundlesLimit int
	//FlushInterval specifies how often the current actualizer flushes changes to underlying storage, optional, default value is 100 milliseconds
	FlushInterval time.Duration

	Broker  in10n.IN10nBroker
	channel in10n.ChannelID
	Opts    []state.ActualizerStateOptFunc
}

type AppStructsFunc func() istructs.IAppStructs

type AsyncActualizerMetrics interface {
	Increase(metricName string, partition istructs.PartitionID, projection appdef.QName, valueDelta float64)
	Set(metricName string, partition istructs.PartitionID, projection appdef.QName, value float64)
}

type SyncActualizerConf struct {
	Ctx          context.Context
	AppStructs   AppStructsFunc
	SecretReader isecrets.ISecretReader
	Partition    istructs.PartitionID
	// Fork responsible for cloning work
	WorkToEvent WorkToEventFunc
	//IntentsLimit top limit per event, default value is 100
	IntentsLimit int
	N10nFunc     state.N10nFunc
}

type IViewSchemaBuilder interface {
	ValueField(name string, kind appdef.DataKind, required bool)
	PartitionKeyField(name string, kind appdef.DataKind, required bool)
	ClusteringColumnField(name string, kind appdef.DataKind, required bool)
}

type BuildViewSchemaFunc func(builder IViewSchemaBuilder)

type WorkToEventFunc func(work interface{}) istructs.IPLogEvent

// AsyncActualizerFactory returns the ServiceOperator<AsyncActualizer>
// workpiece must implement projectors.IAsyncActualizerWork
type AsyncActualizerFactory func(conf AsyncActualizerConf, projection istructs.ProjectorFactory) (pipeline.ISyncOperator, error)

// SyncActualizerFactory returns the Operator<SyncActualizer>
// Workpiece is ...?
type SyncActualizerFactory func(conf SyncActualizerConf, projection istructs.ProjectorFactory, otherProjections ...istructs.ProjectorFactory) pipeline.ISyncOperator
