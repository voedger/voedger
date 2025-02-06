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
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
)

func provideRequestHandler(appParts appparts.IAppPartitions, procbus iprocbus.IProcBus,
	cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpAmount istructs.NumCommandProcessors, vvmApps VVMApps) bus.RequestHandler {
	return func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		if len(request.Resource) <= ShortestPossibleFunctionNameLen {
			bus.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
			return
		}
		funcQName, err := appdef.ParseQName(request.Resource[2:])
		if err != nil {
			bus.ReplyBadRequest(responder, "wrong function name: "+request.Resource)
			return
		}
		if logger.IsVerbose() {
			// FIXME: eliminate this. Unlogged params are logged
			logger.Verbose("request body:\n", string(request.Body))
		}

		appQName, err := appdef.ParseAppQName(request.AppQName)
		if err != nil {
			// protected by router already
			bus.ReplyBadRequest(responder, fmt.Sprintf("failed to parse app qualified name %s: %s", request.AppQName, err.Error()))
			return
		}
		if !vvmApps.Exists(appQName) {
			bus.ReplyBadRequest(responder, "unknown app "+request.AppQName)
			return
		}

		token, err := getPrincipalToken(request)
		if err != nil {
			bus.ReplyAccessDeniedUnauthorized(responder, err.Error())
			return
		}

		partitionID, err := appParts.AppWorkspacePartitionID(appQName, request.WSID)
		if err != nil {
			if errors.Is(err, appparts.ErrNotFound) {
				bus.ReplyErrf(responder, http.StatusServiceUnavailable, fmt.Sprintf("app %s is not deployed", appQName))
				return
			}
			// notest
			bus.ReplyInternalServerError(responder, "failed to compute the partition number", err)
			return
		}

		deliverToProcessors(request, requestCtx, appQName, responder, funcQName, procbus, token, cpchIdx, qpcgIdx, cpAmount, partitionID)
	}
}

func deliverToProcessors(request bus.Request, requestCtx context.Context, appQName appdef.AppQName, responder bus.IResponder, funcQName appdef.QName,
	procbus iprocbus.IProcBus, token string, cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpCount istructs.NumCommandProcessors, partitionID istructs.PartitionID) {
	switch request.Resource[:1] {
	case "q":
		iqm := queryprocessor.NewQueryMessage(requestCtx, appQName, partitionID, request.WSID, responder, request.Body, funcQName, request.Host, token)
		if !procbus.Submit(uint(qpcgIdx), 0, iqm) {
			bus.ReplyErrf(responder, http.StatusServiceUnavailable, "no query processors available")
		}
	case "c":
		// TODO: use appQName to calculate cmdProcessorIdx in solid range [0..cpCount)
		cmdProcessorIdx := uint(partitionID) % uint(cpCount)
		icm := commandprocessor.NewCommandMessage(requestCtx, request.Body, appQName, request.WSID, responder, partitionID, funcQName, token, request.Host)
		if !procbus.Submit(uint(cpchIdx), cmdProcessorIdx, icm) {
			bus.ReplyErrf(responder, http.StatusServiceUnavailable, fmt.Sprintf("command processor of partition %d is busy", partitionID))
		}
	default:
		bus.ReplyBadRequest(responder, fmt.Sprintf(`wrong function mark "%s" for function %s`, request.Resource[:1], funcQName))
	}
}

func getPrincipalToken(request bus.Request) (token string, err error) {
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
