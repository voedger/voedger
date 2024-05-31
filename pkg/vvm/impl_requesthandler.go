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
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
	"github.com/voedger/voedger/staging/src/github.com/untillpro/ibusmem"
)

func provideIBus(appParts appparts.IAppPartitions, procbus iprocbus.IProcBus,
	cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpAmount istructs.NumCommandProcessors, vvmApps VVMApps) ibus.IBus {
	return ibusmem.Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		// Handling Command/Query messages
		// router -> SendRequest2(ctx, ...) -> requestHandler(ctx, ... ) - вот этот контекст. Если connection gracefully closed, то этот ctx.Done()
		// т.е. надо этот контекст пробрасывать далее

		if len(request.Resource) <= ShortestPossibleFunctionNameLen {
			coreutils.ReplyBadRequest(sender, "wrong function name: "+request.Resource)
			return
		}
		funcQName, err := appdef.ParseQName(request.Resource[2:])
		if err != nil {
			coreutils.ReplyBadRequest(sender, "wrong function name: "+request.Resource)
			return
		}
		if logger.IsVerbose() {
			// FIXME: eliminate this. Unlogged params are logged
			logger.Verbose("request body:\n", string(request.Body))
		}

		appQName, err := appdef.ParseAppQName(request.AppQName)
		if err != nil {
			// protected by router already
			coreutils.ReplyBadRequest(sender, fmt.Sprintf("failed to parse app qualified name %s: %s", request.AppQName, err.Error()))
			return
		}
		if !vvmApps.Exists(appQName) {
			coreutils.ReplyBadRequest(sender, fmt.Sprintf("unknown app %s", request.AppQName))
			return
		}

		token, err := getPrincipalToken(request)
		if err != nil {
			coreutils.ReplyAccessDeniedUnauthorized(sender, err.Error())
			return
		}

		partitionID, err := appParts.AppWorkspacePartitionID(appQName, istructs.WSID(request.WSID))
		if err != nil {
			if errors.Is(err, appparts.ErrNotFound) {
				coreutils.ReplyErrf(sender, http.StatusServiceUnavailable, fmt.Sprintf("app %s is not deployed", appQName))
				return
			}
			// notest
			coreutils.ReplyInternalServerError(sender, "failed to compute the partition number", err)
			return
		}

		deliverToProcessors(request, requestCtx, appQName, sender, funcQName, procbus, token, cpchIdx, qpcgIdx, cpAmount, partitionID)
	})
}

func deliverToProcessors(request ibus.Request, requestCtx context.Context, appQName appdef.AppQName, sender ibus.ISender, funcQName appdef.QName,
	procbus iprocbus.IProcBus, token string, cpchIdx CommandProcessorsChannelGroupIdxType, qpcgIdx QueryProcessorsChannelGroupIdxType,
	cpCount istructs.NumCommandProcessors, partitionID istructs.PartitionID) {
	switch request.Resource[:1] {
	case "q":
		iqm := queryprocessor.NewQueryMessage(requestCtx, appQName, istructs.PartitionID(request.PartitionNumber), istructs.WSID(request.WSID), sender, request.Body, funcQName, request.Host, token)
		if !procbus.Submit(int(qpcgIdx), 0, iqm) {
			coreutils.ReplyErrf(sender, http.StatusServiceUnavailable, "no query processors available")
		}
	case "c":
		// TODO: use appQName to calculate cmdProcessorIdx in solid range [0..cpCount)
		cmdProcessorIdx := int64(partitionID) % int64(cpCount)
		icm := commandprocessor.NewCommandMessage(requestCtx, request.Body, appQName, istructs.WSID(request.WSID), sender, partitionID, funcQName, token, request.Host)
		if !procbus.Submit(int(cpchIdx), int(cmdProcessorIdx), icm) {
			coreutils.ReplyErrf(sender, http.StatusServiceUnavailable, fmt.Sprintf("command processor of partition %d is busy", partitionID))
		}
	default:
		coreutils.ReplyBadRequest(sender, fmt.Sprintf(`wrong function mark "%s" for function %s`, request.Resource[:1], funcQName))
	}
}

func getPrincipalToken(request ibus.Request) (token string, err error) {
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

func (rs *resultSenderErrorFirst) checkRS() {
	if rs.rs == nil {
		rs.rs = rs.sender.SendParallelResponse()
	}
}

func (rs *resultSenderErrorFirst) StartArraySection(sectionType string, path []string) {
	rs.checkRS()
	rs.rs.StartArraySection(sectionType, path)
}
func (rs *resultSenderErrorFirst) StartMapSection(sectionType string, path []string) {
	rs.checkRS()
	rs.rs.StartMapSection(sectionType, path)
}
func (rs *resultSenderErrorFirst) ObjectSection(sectionType string, path []string, element interface{}) (err error) {
	rs.checkRS()
	return rs.rs.ObjectSection(sectionType, path, element)
}
func (rs *resultSenderErrorFirst) SendElement(name string, element interface{}) (err error) {
	rs.checkRS()
	return rs.rs.SendElement(name, element)
}
func (rs *resultSenderErrorFirst) Close(err error) {
	defer func() {
		if err != nil {
			logger.Error(err)
		}
	}()
	if rs.rs != nil {
		rs.rs.Close(err)
		return
	}
	if err != nil {
		coreutils.ReplyErr(rs.sender, err)
		return
	}
	coreutils.ReplyJSON(rs.sender, http.StatusOK, "{}")
}
