/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
)

func provideRequestHandler(appParts appparts.IAppPartitions, procbus iprocbus.IProcBus,
	cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpAmount istructs.NumCommandProcessors, vvmApps VVMApps) coreutils.RequestHandler {
	return func(requestCtx context.Context, request coreutils.Request, responder coreutils.IResponder) {
		if len(request.Resource) <= ShortestPossibleFunctionNameLen {
			coreutils.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
			return
		}
		funcQName, err := appdef.ParseQName(request.Resource[2:])
		if err != nil {
			coreutils.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
			return
		}
		if logger.IsVerbose() {
			// FIXME: eliminate this. Unlogged params are logged
			logger.Verbose("request body:\n", string(request.Body))
		}

		appQName, err := appdef.ParseAppQName(request.AppQName)
		if err != nil {
			// protected by router already
			coreutils.ReplyBadRequest(responder, fmt.Sprintf("failed to parse app qualified name %s: %s", request.AppQName, err.Error()))
			return
		}
		if !vvmApps.Exists(appQName) {
			coreutils.ReplyBadRequest(responder, "unknown app "+request.AppQName)
			return
		}

		token, err := getPrincipalToken(request)
		if err != nil {
			coreutils.ReplyAccessDeniedUnauthorized(responder, err.Error())
			return
		}

		partitionID, err := appParts.AppWorkspacePartitionID(appQName, request.WSID)
		if err != nil {
			if errors.Is(err, appparts.ErrNotFound) {
				coreutils.ReplyErrf(responder, http.StatusServiceUnavailable, fmt.Sprintf("app %s is not deployed", appQName))
				return
			}
			// notest
			coreutils.ReplyInternalServerError(responder, "failed to compute the partition number", err)
			return
		}

		deliverToProcessors(request, requestCtx, appQName, responder, funcQName, procbus, token, cpchIdx, qpcgIdx, cpAmount, partitionID)
	}
}

// func provideIBus(appParts appparts.IAppPartitions, procbus iprocbus.IProcBus,
// 	cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
// 	cpAmount istructs.NumCommandProcessors, vvmApps VVMApps) ibus.IBus {
// 	return ibusmem.Provide(func(requestCtx context.Context, replier coreutils.IReplier, request coreutils.Request) {
// 		// Handling Command/Query messages
// 		// router -> SendRequest2(ctx, ...) -> requestHandler(ctx, ... ) - it is that context. If connection gracefully closed the that ctx is Done()
// 		// so we need to forward that context

// 		if len(request.Resource) <= ShortestPossibleFunctionNameLen {
// 			coreutils.ReplyBadRequest(replier, "wrong function name: "+request.Resource)
// 			return
// 		}
// 		funcQName, err := appdef.ParseQName(request.Resource[2:])
// 		if err != nil {
// 			coreutils.ReplyBadRequest(replier, "wrong function name: "+request.Resource)
// 			return
// 		}
// 		if logger.IsVerbose() {
// 			// FIXME: eliminate this. Unlogged params are logged
// 			logger.Verbose("request body:\n", string(request.Body))
// 		}

// 		appQName, err := appdef.ParseAppQName(request.AppQName)
// 		if err != nil {
// 			// protected by router already
// 			coreutils.ReplyBadRequest(replier, fmt.Sprintf("failed to parse app qualified name %s: %s", request.AppQName, err.Error()))
// 			return
// 		}
// 		if !vvmApps.Exists(appQName) {
// 			coreutils.ReplyBadRequest(replier, "unknown app "+request.AppQName)
// 			return
// 		}

// 		token, err := getPrincipalToken(request)
// 		if err != nil {
// 			coreutils.ReplyAccessDeniedUnauthorized(replier, err.Error())
// 			return
// 		}

// 		partitionID, err := appParts.AppWorkspacePartitionID(appQName, request.WSID)
// 		if err != nil {
// 			if errors.Is(err, appparts.ErrNotFound) {
// 				coreutils.ReplyErrf(replier, http.StatusServiceUnavailable, fmt.Sprintf("app %s is not deployed", appQName))
// 				return
// 			}
// 			// notest
// 			coreutils.ReplyInternalServerError(replier, "failed to compute the partition number", err)
// 			return
// 		}

// 		deliverToProcessors(request, requestCtx, appQName, replier, funcQName, procbus, token, cpchIdx, qpcgIdx, cpAmount, partitionID)
// 	})
// }

func deliverToProcessors(request coreutils.Request, requestCtx context.Context, appQName appdef.AppQName, responder coreutils.IResponder, funcQName appdef.QName,
	procbus iprocbus.IProcBus, token string, cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpCount istructs.NumCommandProcessors, partitionID istructs.PartitionID) {
	switch request.Resource[:1] {
	case "q":
		iqm := queryprocessor.NewQueryMessage(requestCtx, appQName, request.PartitionID, request.WSID, responder, request.Body, funcQName, request.Host, token)
		if !procbus.Submit(uint(qpcgIdx), 0, iqm) {
			coreutils.ReplyErrf(responder, http.StatusServiceUnavailable, "no query processors available")
		}
	case "c":
		// TODO: use appQName to calculate cmdProcessorIdx in solid range [0..cpCount)
		cmdProcessorIdx := uint(partitionID) % uint(cpCount)
		icm := commandprocessor.NewCommandMessage(requestCtx, request.Body, appQName, request.WSID, responder, partitionID, funcQName, token, request.Host)
		if !procbus.Submit(uint(cpchIdx), cmdProcessorIdx, icm) {
			coreutils.ReplyErrf(responder, http.StatusServiceUnavailable, fmt.Sprintf("command processor of partition %d is busy", partitionID))
		}
	default:
		coreutils.ReplyBadRequest(responder, fmt.Sprintf(`wrong function mark "%s" for function %s`, request.Resource[:1], funcQName))
	}
}

func getPrincipalToken(request coreutils.Request) (token string, err error) {
	authHeaders := request.Header[coreutils.Authorization]
	if len(authHeaders) == 0 {
		return "", nil
	}
	authHeader := authHeaders[0]
	if strings.HasPrefix(authHeader, coreutils.BearerPrefix) {
		return strings.ReplaceAll(authHeader, coreutils.BearerPrefix, ""), nil
	}
	if strings.HasPrefix(authHeader, "Basic ") {
		return getBasicAuthToken(authHeader)
	}
	return "", errors.New("unsupported Authorization header: " + authHeader)
}

func getBasicAuthToken(authHeader string) (token string, err error) {
	headerValue := strings.ReplaceAll(authHeader, "Basic ", "")
	headerValueBytes, err := base64.StdEncoding.DecodeString(headerValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 Basic Authorization header value: %w", err)
	}
	headerValue = string(headerValueBytes)
	if strings.Count(headerValue, ":") != 1 {
		return "", errors.New("unexpected Basic Authorization header format")
	}
	return strings.ReplaceAll(headerValue, ":", ""), nil
}
