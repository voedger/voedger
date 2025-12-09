/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"context"
	"net/url"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIFederation struct {
	httpClient             httpu.IHTTPClient
	federationURL          func() *url.URL
	adminPortGetter        func() int
	defaultReqOptFuncs     []httpu.ReqOptFunc
	vvmCtx                 context.Context
	policyOptsForWithRetry PolicyOptsForWithRetry
}

type OffsetsChan chan istructs.Offset

// implements json.Unmarshaler
type CommandResponse struct {
	NewIDs            map[string]istructs.RecordID
	CurrentWLogOffset istructs.Offset
	CmdResult         map[string]interface{}
}

type QueryResponse struct {
	QPv2Response // TODO: eliminate this after https://github.com/voedger/voedger/issues/1313
	Sections     []struct {
		Elements [][][][]interface{} `json:"elements"`
	} `json:"sections"`
}

type QPv2Response []map[string]interface{}

func (r QPv2Response) Result() map[string]interface{} {
	return r.ResultRow(0)
}

func (r QPv2Response) ResultRow(rowNum int) map[string]interface{} {
	return r[rowNum]
}

// implements json.Unmarshaler
type FuncResponse struct {
	httpu.HTTPResponse
	CommandResponse
	QueryResponse
	SysError error
}

type PolicyOptsForWithRetry []httpu.RetryPolicyOpt
