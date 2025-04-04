/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"context"

	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type queryProcessorState struct {
	*hostState
	queryArgs          state.PrepareArgsFunc
	queryCallback      state.ExecQueryCallbackFunc
	resultValueBuilder istructs.IStateValueBuilder
}

func (s queryProcessorState) QueryPrepareArgs() istructs.PrepareArgs {
	return s.queryArgs()
}

func (s queryProcessorState) QueryCallback() istructs.ExecQueryCallback {
	return s.queryCallback()
}

func (s *queryProcessorState) sendPrevQueryObject() error {
	if s.resultValueBuilder != nil {
		obj := s.resultValueBuilder.BuildValue().(*storages.ObjectStateValue).AsObject()
		s.resultValueBuilder = nil
		return s.queryCallback()(obj)
	}
	return nil
}

func (s *queryProcessorState) NewValue(key istructs.IStateKeyBuilder) (eb istructs.IStateValueBuilder, err error) {
	if key.Storage() == sys.Storage_Result {
		err = s.sendPrevQueryObject()
		if err != nil {
			return nil, err
		}
		eb, err = s.hostState.withInsert[sys.Storage_Result].ProvideValueBuilder(key, nil)
		if err != nil {
			return nil, err
		}
		s.resultValueBuilder = eb
		return eb, nil
	}
	return s.hostState.NewValue(key)
}

func (s *queryProcessorState) FindIntent(key istructs.IStateKeyBuilder) istructs.IStateValueBuilder {
	if key.Storage() == sys.Storage_Result {
		return s.resultValueBuilder
	}
	return s.hostState.FindIntent(key)
}

func (s *queryProcessorState) ApplyIntents() (err error) {
	err = s.sendPrevQueryObject()
	if err != nil {
		return err
	}
	return s.hostState.ApplyIntents()
}

func implProvideQueryProcessorState(
	ctx context.Context,
	appStructsFunc state.AppStructsFunc,
	partitionIDFunc state.PartitionIDFunc,
	wsidFunc state.WSIDFunc,
	secretReader isecrets.ISecretReader,
	principalsFunc state.PrincipalsFunc,
	tokenFunc state.TokenFunc,
	itokens itokens.ITokens,
	execQueryArgsFunc state.PrepareArgsFunc,
	argFunc state.ArgFunc,
	resultBuilderFunc state.ObjectBuilderFunc,
	federation federation.IFederation,
	queryCallbackFunc state.ExecQueryCallbackFunc,
	options ...state.StateOptFunc) state.IHostState {

	opts := &state.StateOpts{}
	for _, optFunc := range options {
		optFunc(opts)
	}

	state := &queryProcessorState{
		hostState:     newHostState("QueryProcessor", queryProcessorStateMaxIntents, appStructsFunc),
		queryArgs:     execQueryArgsFunc,
		queryCallback: queryCallbackFunc,
	}

	ieventsFunc := func() istructs.IEvents {
		return appStructsFunc().Events()
	}

	state.addStorage(sys.Storage_View, storages.NewViewRecordsStorage(ctx, appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH|S_READ)
	state.addStorage(sys.Storage_Record, storages.NewRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	state.addStorage(sys.Storage_WLog, storages.NewWLogStorage(ctx, ieventsFunc, wsidFunc), S_GET|S_READ)
	state.addStorage(sys.Storage_HTTP, storages.NewHTTPStorage(opts.CustomHTTPClient), S_READ)
	state.addStorage(sys.Storage_FederationCommand, storages.NewFederationCommandStorage(appStructsFunc, wsidFunc, federation, itokens, opts.FederationCommandHandler), S_GET)
	state.addStorage(sys.Storage_FederationBlob, storages.NewFederationBlobStorage(appStructsFunc, wsidFunc, federation, itokens, opts.FederationBlobHandler), S_READ)
	state.addStorage(sys.Storage_AppSecret, storages.NewAppSecretsStorage(secretReader), S_GET)
	state.addStorage(sys.Storage_RequestSubject, storages.NewSubjectStorage(principalsFunc, tokenFunc), S_GET)
	state.addStorage(sys.Storage_QueryContext, storages.NewQueryContextStorage(argFunc, wsidFunc), S_GET)
	state.addStorage(sys.Storage_Response, storages.NewResponseStorage(), S_INSERT)
	state.addStorage(sys.Storage_Result, storages.NewResultStorage(resultBuilderFunc), S_INSERT)
	state.addStorage(sys.Storage_Uniq, storages.NewUniquesStorage(appStructsFunc, wsidFunc, opts.UniquesHandler), S_GET)
	state.addStorage(sys.Storage_Logger, storages.NewLoggerStorage(), S_INSERT)

	return state
}
