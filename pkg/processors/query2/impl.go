/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func implServiceFactory(serviceChannel iprocbus.ServiceChannel,
	appParts appparts.IAppPartitions, maxPrepareQueries int, metrics imetrics.IMetrics, vvm string,
	authn iauthnz.IAuthenticator, itokens itokens.ITokens, federation federation.IFederation,
	statelessResources istructsmem.IStatelessResources, secretReader isecrets.ISecretReader) pipeline.IService {
	return pipeline.NewService(func(ctx context.Context) {
		var p pipeline.ISyncPipeline
		for ctx.Err() == nil {
			select {
			case intf := <-serviceChannel:
				now := time.Now()
				msg := intf.(IQueryMessage)
				qpm := &queryProcessorMetrics{
					vvm:     vvm,
					app:     msg.AppQName(),
					metrics: metrics,
				}
				qpm.Increase(queryprocessor.Metric_QueriesTotal, 1.0)
				qwork := newQueryWork(msg, appParts, maxPrepareQueries, qpm, secretReader)
				func() { // borrowed application partition should be guaranteed to be freed
					defer qwork.Release()
					if p == nil {
						p = newQueryProcessorPipeline(ctx, authn, itokens, federation, statelessResources)
					}
					err := p.SendSync(qwork)
					if err != nil {
						qpm.Increase(queryprocessor.Metric_ErrorsTotal, 1.0)
						p.Close()
						p = nil
					} else {
						/* TODO: implement
							if err = execQuery(ctx, qwork); err == nil {
							if err = processors.CheckResponseIntent(qwork.state); err == nil {
								err = qwork.state.ApplyIntents()
							}
						}
						*/
					}
					if qwork.rowsProcessor != nil {
						// wait until all rows are sent
						qwork.rowsProcessor.Close()
					}
					select {
					case rowsProcErr := <-qwork.rowsProcessorErrCh:
						if err == nil {
							err = rowsProcErr
						}
					default:
					}
					err = coreutils.WrapSysError(err, http.StatusInternalServerError)
					var senderCloseable bus.IResponseSenderCloseable
					statusCode := http.StatusOK
					if err != nil {
						statusCode = err.(coreutils.SysError).HTTPStatus // nolint:errorlint
					}
					if qwork.responseSenderGetter == nil || qwork.responseSenderGetter() == nil {
						// have an error before 200ok is sent -> send the status from the actual error
						senderCloseable = msg.Responder().InitResponse(bus.ResponseMeta{
							ContentType: coreutils.ApplicationJSON,
							StatusCode:  statusCode,
						})
					} else {
						sender := qwork.responseSenderGetter()
						senderCloseable = sender.(bus.IResponseSenderCloseable)
					}
					senderCloseable.Close(err)
				}()
				metrics.IncreaseApp(queryprocessor.Metric_QueriesSeconds, vvm, msg.AppQName(), time.Since(now).Seconds())
			case <-ctx.Done():
			}
		}
		if p != nil {
			p.Close()
		}
	})
}

// IStatelessResources need only for determine the exact result type of ANY
func newQueryProcessorPipeline(requestCtx context.Context, authn iauthnz.IAuthenticator,
	itokens itokens.ITokens, federation federation.IFederation, statelessResources istructsmem.IStatelessResources) pipeline.ISyncPipeline {
	ops := []*pipeline.WiredOperator{
		operator("borrowAppPart", borrowAppPart),
		operator("check rate", func(ctx context.Context, qw *queryWork) (err error) {
			switch qw.msg.ApiPath() {
			case ApiPath_Queries:
				if qw.appStructs.IsFunctionRateLimitsExceeded(qw.msg.QName(), qw.msg.WSID()) {
					return coreutils.NewSysError(http.StatusTooManyRequests)
				}
				break
			case ApiPath_Docs:
			case ApiPath_CDocs:
			case ApiPath_Views:
				// TODO: implement rate limits for other paths
				break
			}
			return nil
		}),
		operator("authenticate query request", func(ctx context.Context, qw *queryWork) (err error) {
			req := iauthnz.AuthnRequest{
				Host:        qw.msg.Host(),
				RequestWSID: qw.msg.WSID(),
				Token:       qw.msg.Token(),
			}
			if qw.principals, qw.principalPayload, err = authn.Authenticate(qw.msg.RequestCtx(), qw.appStructs, qw.appStructs.AppTokens(), req); err != nil {
				return coreutils.WrapSysError(err, http.StatusUnauthorized)
			}
			return
		}),
		operator("get workspace descriptor", func(ctx context.Context, qw *queryWork) (err error) {
			qw.wsDesc, err = qw.appStructs.Records().GetSingleton(qw.msg.WSID(), authnz.QNameCDocWorkspaceDescriptor)
			return err
		}),
		operator("check cdoc.sys.WorkspaceDescriptor existence", func(ctx context.Context, qw *queryWork) (err error) {
			if qw.wsDesc.QName() == appdef.NullQName {
				// TODO: ws init check is simplified here because we need just IWorkspace to get the query from it.
				return processors.ErrWSNotInited
			}
			return nil
		}),
		operator("get principals roles", func(ctx context.Context, qw *queryWork) (err error) {
			for _, prn := range qw.principals {
				if prn.Kind != iauthnz.PrincipalKind_Role {
					continue
				}
				qw.roles = append(qw.roles, prn.QName)
			}
			return nil
		}),
		operator("check workspace active", func(ctx context.Context, qw *queryWork) (err error) {
			for _, prn := range qw.principals {
				if prn.Kind == iauthnz.PrincipalKind_Role && prn.QName == iauthnz.QNameRoleSystem && prn.WSID == qw.msg.WSID() {
					// system -> allow to work in any case
					return nil
				}
			}
			if qw.wsDesc.QName() == appdef.NullQName {
				// TODO: query prcessor currently does not check the workspace active state
				return nil
			}
			if qw.wsDesc.AsInt32(authnz.Field_Status) != int32(authnz.WorkspaceStatus_Active) {
				return processors.ErrWSInactive
			}
			return nil
		}),

		operator("get IWorkspace", func(ctx context.Context, qw *queryWork) (err error) {
			if qw.wsDesc.QName() == appdef.NullQName {
				// workspace is dummy
				return nil
			}
			if qw.iWorkspace = qw.appStructs.AppDef().WorkspaceByDescriptor(qw.wsDesc.AsQName(authnz.Field_WSKind)); qw.iWorkspace == nil {
				return coreutils.NewHTTPErrorf(http.StatusInternalServerError, fmt.Sprintf("workspace is not found in AppDef by cdoc.sys.WorkspaceDescriptor.WSKind %s",
					qw.wsDesc.AsQName(authnz.Field_WSKind)))
			}
			return nil
		}),

		operator("get IQuery", func(ctx context.Context, qw *queryWork) (err error) {
			switch qw.msg.ApiPath() {
			case ApiPath_Queries:
				switch qw.iWorkspace {
				case nil:
					// workspace is dummy
					if qw.iQuery = appdef.Query(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iQuery == nil {
						return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist", qw.msg.QName()))
					}
				default:
					if qw.iQuery = appdef.Query(qw.iWorkspace.Type, qw.msg.QName()); qw.iQuery == nil {
						return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
					}
				}
			case ApiPath_Views:
			case ApiPath_Docs:
			case ApiPath_CDocs:
				// TODO: implement
			}
			return nil
		}),

		operator("authorize query request", func(ctx context.Context, qw *queryWork) (err error) {
			// TODO: implement
			/*
				ws := qw.iWorkspace
				if ws == nil {
					// workspace is dummy
					ws = qw.iQuery.Workspace()
				}
				ok, _, err := qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Execute, qw.msg.QName(), nil, qw.roles)
				if err != nil {
					return err
				}
				if !ok {
					return coreutils.WrapSysError(errors.New(""), http.StatusForbidden)
				};*/
			return nil
		}),
		operator("unmarshal request", func(ctx context.Context, qw *queryWork) (err error) {
			/* TODO: implement
			parsType := qw.iQuery.Param()
			if parsType != nil && parsType.QName() == istructs.QNameRaw {
				qw.requestData["args"] = map[string]interface{}{
					processors.Field_RawObject_Body: string(qw.msg.Body()),
				}
				return nil
			}
			err = coreutils.JSONUnmarshal(qw.msg.Body(), &qw.requestData)\*/
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("validate: get exec query args", func(ctx context.Context, qw *queryWork) (err error) {
			// TODO: implement
			// 	qw.execQueryArgs, err = newExecQueryArgs(qw.requestData, qw.msg.WSID(), qw)
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("create callback func", func(ctx context.Context, qw *queryWork) (err error) {
			/* TODO: implement
			qw.callbackFunc = func(object istructs.IObject) error {
				pathToIdx := make(map[string]int)
				if qw.resultType.QName() == istructs.QNameRaw {
					pathToIdx[processors.Field_RawObject_Body] = 0
				} else {
					for i, element := range qw.queryParams.Elements() {
						pathToIdx[element.Path().Name()] = i
					}
				}
				return qw.rowsProcessor.SendAsync(rowsWorkpiece{
					object: object,
					outputRow: &outputRow{
						keyToIdx: pathToIdx,
						values:   make([]interface{}, len(pathToIdx)),
					},
					enrichedRootFieldsKinds: make(map[string]appdef.DataKind),
				})
			}*/
			return nil
		}),
		operator("create state", func(ctx context.Context, qw *queryWork) (err error) {
			qw.state = stateprovide.ProvideQueryProcessorStateFactory()(
				qw.msg.RequestCtx(),
				func() istructs.IAppStructs { return qw.appStructs },
				state.SimplePartitionIDFunc(qw.msg.Partition()),
				state.SimpleWSIDFunc(qw.msg.WSID()),
				qw.secretReader,
				func() []iauthnz.Principal { return qw.principals },
				func() string { return qw.msg.Token() },
				itokens,
				func() istructs.PrepareArgs { return qw.execQueryArgs.PrepareArgs },
				func() istructs.IObject { return qw.execQueryArgs.ArgumentObject },
				func() istructs.IObjectBuilder {
					return qw.appStructs.ObjectBuilder(qw.resultType.QName())
				},
				federation,
				func() istructs.ExecQueryCallback {
					return qw.callbackFunc
				},
			)
			qw.execQueryArgs.State = qw.state
			qw.execQueryArgs.Intents = qw.state
			return
		}),
		operator("validate: get result type", func(ctx context.Context, qw *queryWork) (err error) {
			/* TODO: implement
			qw.resultType = qw.iQuery.Result()
			if qw.resultType == nil || qw.resultType.QName() != appdef.QNameANY {
				return nil
			}
			// ANY -> exact type according to PrepareArgs
			iResource := qw.appStructs.Resources().QueryResource(qw.msg.QName())
			var iQueryFunc istructs.IQueryFunction
			if iResource.Kind() != istructs.ResourceKind_null {
				iQueryFunc = iResource.(istructs.IQueryFunction)
			} else {
				for _, qry := range statelessResources.Queries {
					if qry.QName() == qw.msg.QName() {
						iQueryFunc = qry
						break
					}
				}
			}
			qNameResultType := iQueryFunc.ResultType(qw.execQueryArgs.PrepareArgs)

			ws := qw.iWorkspace
			if ws == nil {
				// workspace is dummy
				ws = qw.iQuery.Workspace()
			}
			qw.resultType = ws.Type(qNameResultType)
			if qw.resultType.Kind() == appdef.TypeKind_null {
				return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%s query result type %s does not exist in %v", qw.iQuery.QName(), qNameResultType, ws))
			}*/
			return nil
		}),
		operator("validate: get query params", func(ctx context.Context, qw *queryWork) (err error) {
			/* TODO: implement
			qw.queryParams, err = newQueryParams(qw.requestData, NewElement, NewFilter, NewOrderBy, newFieldsKinds(qw.resultType), qw.resultType)
			*/
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("authorize actual sys.Any result", func(ctx context.Context, qw *queryWork) (err error) {
			/* TODO: implement
			if qw.iQuery.Result() != appdef.AnyType {
				// will authorize result only if result is sys.Any
				// otherwise each field is considered as allowed if EXECUTE ON QUERY is allowed
				return nil
			}
			ws := qw.iWorkspace
			if ws == nil {
				// workspace is dummy
				ws = qw.iQuery.Workspace()
			}
			for _, elem := range qw.queryParams.Elements() {
				nestedPath := elem.Path().AsArray()
				nestedType := qw.resultType
				for _, nestedName := range nestedPath {
					if len(nestedName) == 0 {
						// root
						continue
					}
					// incorrectness is excluded already on validation stage in [queryParams.validate]
					containersOfNested := nestedType.(appdef.IWithContainers)
					// container presence is checked already on validation stage in [queryParams.validate]
					nestedContainer := containersOfNested.Container(nestedName)
					nestedType = nestedContainer.Type()
				}
				requestedfields := []string{}
				for _, resultField := range elem.ResultFields() {
					requestedfields = append(requestedfields, resultField.Field())
				}
				ok, allowedFields, err := qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, nestedType.QName(), requestedfields, qw.roles)
				if err != nil {
					return err
				}
				if !ok {
					return coreutils.NewSysError(http.StatusForbidden)
				}
				for _, requestedField := range requestedfields {
					if !slices.Contains(allowedFields, requestedField) {
						return coreutils.NewSysError(http.StatusForbidden)
					}
				}
			}
			*/
			return nil
		}),

		operator("build rows processor", func(ctx context.Context, qw *queryWork) error {
			now := time.Now()
			defer func() {
				qw.metrics.Increase(queryprocessor.Metric_BuildSeconds, time.Since(now).Seconds())
			}()
			/*
				TODO: implement
				qw.rowsProcessor, qw.responseSenderGetter = ProvideRowsProcessorFactory()(qw.msg.RequestCtx(), qw.appStructs.AppDef(),
					qw.state, qw.queryParams, qw.resultType, qw.msg.Responder(), qw.metrics, qw.rowsProcessorErrCh)
			*/
			return nil
		}),
	}
	return pipeline.NewSyncPipeline(requestCtx, "Query Processor", ops[0], ops[1:]...)
}
