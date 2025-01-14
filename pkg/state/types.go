/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"io"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state/smtptest"

	"github.com/voedger/voedger/pkg/coreutils/federation"
)

type PartitionIDFunc func() istructs.PartitionID
type WSIDFunc func() istructs.WSID
type N10nFunc func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset)
type AppStructsFunc func() istructs.IAppStructs
type CUDFunc func() istructs.ICUD
type ObjectBuilderFunc func() istructs.IObjectBuilder
type PrincipalsFunc func() []iauthnz.Principal
type TokenFunc func() string
type PLogEventFunc func() istructs.IPLogEvent
type CommandPrepareArgsFunc func() istructs.CommandPrepareArgs
type ArgFunc func() istructs.IObject
type UnloggedArgFunc func() istructs.IObject
type WLogOffsetFunc func() istructs.Offset
type FederationFunc func() federation.IFederation
type QNameFunc func() appdef.QName
type TokensFunc func() itokens.ITokens
type PrepareArgsFunc func() istructs.PrepareArgs
type ExecQueryCallbackFunc func() istructs.ExecQueryCallback
type UnixTimeFunc func() int64
type MockedStateFactory func(intentsLimit int, appStructsFunc AppStructsFunc) IHostState
type CommandProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, intentsLimit int, cmdResultBuilderFunc ObjectBuilderFunc, execCmdArgsFunc CommandPrepareArgsFunc, argFunc ArgFunc, unloggedArgFunc UnloggedArgFunc, wlogOffsetFunc WLogOffsetFunc, opts ...StateOptFunc) IHostState
type SyncActualizerStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit int, opts ...StateOptFunc) IHostState
type QueryProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, itokens itokens.ITokens, execQueryArgsFunc PrepareArgsFunc, argFunc ArgFunc, resultBuilderFunc ObjectBuilderFunc, federation federation.IFederation, queryCallbackFunc ExecQueryCallbackFunc, opts ...StateOptFunc) IHostState
type AsyncActualizerStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, tokensFunc itokens.ITokens, federationFunc federation.IFederation, intentsLimit, bundlesLimit int, opts ...StateOptFunc) IBundledHostState
type SchedulerStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, tokensFunc itokens.ITokens, federationFunc federation.IFederation, unixTimeFunc UnixTimeFunc, intentsLimit int, optFuncs ...StateOptFunc) IHostState

type FederationCommandHandler = func(owner, appname string, wsid istructs.WSID, command appdef.QName, body string) (statusCode int, newIDs map[string]istructs.RecordID, result string, err error)
type FederationBlobHandler = func(owner, appname string, wsid istructs.WSID, blobId istructs.RecordID) (result []byte, err error)
type UniquesHandler = func(entity appdef.QName, wsid istructs.WSID, data map[string]interface{}) (istructs.RecordID, error)

type EventsFunc func() istructs.IEvents
type RecordsFunc func() istructs.IRecords

type StateOptFunc func(opts *StateOpts)

type IHttpClient interface {
	Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error)
}

type StateOpts struct {
	Messages                 chan smtptest.Message
	FederationCommandHandler FederationCommandHandler
	FederationBlobHandler    FederationBlobHandler
	CustomHttpClient         IHttpClient
	UniquesHandler           UniquesHandler
}

func WithEmailMessagesChan(messages chan smtptest.Message) StateOptFunc {
	return func(opts *StateOpts) {
		opts.Messages = messages
	}
}

func WithCustomHttpClient(client IHttpClient) StateOptFunc {
	return func(opts *StateOpts) {
		opts.CustomHttpClient = client
	}
}

func WithFedearationCommandHandler(handler FederationCommandHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.FederationCommandHandler = handler
	}
}

func WithFederationBlobHandler(handler FederationBlobHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.FederationBlobHandler = handler
	}
}

func WithUniquesHandler(handler UniquesHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.UniquesHandler = handler
	}
}

type ApplyBatchItem struct {
	Key   istructs.IStateKeyBuilder
	Value istructs.IStateValueBuilder
	IsNew bool
}

type GetBatchItem struct {
	Key   istructs.IStateKeyBuilder
	Value istructs.IStateValue
}
