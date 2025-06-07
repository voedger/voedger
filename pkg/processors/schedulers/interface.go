/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/state"
)

type TimeAfterFunc func(d time.Duration) <-chan time.Time

type LogErrorFunc func(args ...interface{})

type BasicSchedulerConfig struct {
	VvmName processors.VVMName

	SecretReader isecrets.ISecretReader
	Tokens       itokens.ITokens
	Metrics      imetrics.IMetrics
	Broker       in10n.IN10nBroker
	Federation   federation.IFederation
	Time         timeu.ITime

	Opts []state.StateOptFunc

	// Optional. Default value: `time.After`
	AfterError TimeAfterFunc
	// Optional. Default value: `core-logger.Error`
	LogError LogErrorFunc
	//IntentsLimit top limit per event, optional, default value is 100
	IntentsLimit int
}

type SchedulerConfig struct {
	BasicSchedulerConfig

	AppQName  appdef.AppQName
	Workspace istructs.WSID
	AppWSIdx  istructs.AppWorkspaceNumber
	Partition istructs.PartitionID // ?
}

type ISchedulersService interface {
	pipeline.IServiceEx
	appparts.ISchedulerRunner
}
