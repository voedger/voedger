/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/processors/n10n"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	"github.com/voedger/voedger/pkg/processors/query2"
)

func provideRequestHandler(appParts appparts.IAppPartitions, procbus iprocbus.IProcBus,
	cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx_v1 QueryProcessorsChannelGroupIdxType_V1,
	qpcgIdx_v2 QueryProcessorsChannelGroupIdxType_V2,
	cpAmount istructs.NumCommandProcessors, vvmApps VVMApps, n10nProc n10n.IN10NProc,
	busyLogMode BusyProcessorLogMode) bus.RequestHandler {
	return func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		token, err := bus.GetPrincipalToken(request)
		if err != nil {
			bus.ReplyAccessDeniedUnauthorized(responder, err.Error())
			return
		}
		if request.IsN10N {
			n10nArgs := n10n.N10NProcArgs{
				Host:             request.Host,
				Body:             request.Body,
				Token:            token,
				Method:           request.Method,
				EntityFromURL:    request.QName,
				WSID:             request.WSID,
				Responder:        responder,
				AppQName:         request.AppQName,
				ChannelIDFromURL: request.Resource,
			}
			// This Handle call does not block even if the n10n processor is busy with a previous long-polling request,
			// because a new pipeline is created for each request.
			// The call is made directly instead of via procbus; n10n concurrency is already limited by the number of
			// subscriptions per subject and per login.
			n10nProc.Handle(requestCtx, n10nArgs)
			return
		}

		if !vvmApps.Exists(request.AppQName) {
			bus.ReplyBadRequest(responder, "unknown app "+request.AppQName.String())
			return
		}

		partitionID, err := appParts.AppWorkspacePartitionID(request.AppQName, request.WSID)
		if err != nil {
			if errors.Is(err, appparts.ErrNotFound) {
				bus.ReplyErrf(responder, http.StatusServiceUnavailable, fmt.Sprintf("app %s is not deployed", request.AppQName))
				return
			}
			// notest
			bus.ReplyInternalServerError(responder, "failed to compute the partition number", err)
			return
		}

		// deliver to processors
		if request.IsAPIV2 {
			if request.Method == http.MethodGet {
				// QP
				iqm := query2.NewIQueryMessage(requestCtx, request.AppQName, request.WSID, responder, request.Query, request.DocID, processors.APIPath(request.APIPath), request.QName,
					partitionID, request.Host, token, request.WorkspaceQName, request.Header[httpu.Accept])
				if !procbus.Submit(uint(qpcgIdx_v2), 0, iqm) {
					replyQueryBusy(requestCtx, request.IsAPIV2, responder, busyLogMode)
				}
			} else {
				// CP

				// TODO: use appQName to calculate cmdProcessorIdx in solid range [0..cpCount)
				cmdProcessorIdx := uint(partitionID) % uint(cpAmount)
				icm := commandprocessor.NewCommandMessage(requestCtx, request.Body, request.AppQName, request.WSID, responder, partitionID, request.QName, token,
					request.Host, processors.APIPath(request.APIPath), istructs.RecordID(request.DocID), request.Method, request.Header[httpu.Origin])
				if !procbus.Submit(uint(cpchIdx), cmdProcessorIdx, icm) {
					replyCommandBusy(requestCtx, responder, partitionID, busyLogMode)
				}
			}
		} else {
			if len(request.Resource) <= ShortestPossibleFunctionNameLen {
				bus.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
				return
			}
			funcQName, err := appdef.ParseQName(request.Resource[2:])
			if err != nil {
				bus.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
				return
			}

			switch request.Resource[:1] {
			case "q":
				iqm := queryprocessor.NewQueryMessage(requestCtx, request.AppQName, partitionID, request.WSID, responder, request.Body, funcQName, request.Host, token)
				if !procbus.Submit(uint(qpcgIdx_v1), 0, iqm) {
					replyQueryBusy(requestCtx, request.IsAPIV2, responder, busyLogMode)
				}
			case "c":
				// TODO: use appQName to calculate cmdProcessorIdx in solid range [0..cpCount)
				cmdProcessorIdx := uint(partitionID) % uint(cpAmount)
				icm := commandprocessor.NewCommandMessage(requestCtx, request.Body, request.AppQName, request.WSID, responder, partitionID, funcQName, token,
					request.Host, processors.APIPath(request.APIPath), istructs.RecordID(request.DocID), request.Method, request.Header[httpu.Origin])
				if !procbus.Submit(uint(cpchIdx), cmdProcessorIdx, icm) {
					replyCommandBusy(requestCtx, responder, partitionID, busyLogMode)
				}
			default:
				bus.ReplyBadRequest(responder, fmt.Sprintf(`wrong function mark "%s" for function %s`, request.Resource[:1], funcQName))
			}
		}
	}
}

func replyQueryBusy(ctx context.Context, isAPIv2 bool, responder bus.IResponder, logMode BusyProcessorLogMode) {
	str := "v1"
	if isAPIv2 {
		str = "v2"
	}
	msg := fmt.Sprintf("no query processors %s available", str)
	if logMode == BusyProcessorLogMode_Error {
		logger.ErrorCtx(ctx, "vvm.submit", msg)
	}
	bus.ReplyErrf(responder, http.StatusServiceUnavailable, msg)
}

func replyCommandBusy(ctx context.Context, responder bus.IResponder, partitionID istructs.PartitionID, logMode BusyProcessorLogMode) {
	msg := fmt.Sprintf("no command processors available, partition %d", partitionID)
	if logMode == BusyProcessorLogMode_Error {
		logger.ErrorCtx(ctx, "vvm.submit", msg)
	}
	bus.ReplyErrf(responder, http.StatusServiceUnavailable, msg)
}
