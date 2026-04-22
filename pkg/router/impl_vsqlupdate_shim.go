/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/processors"
)

const (
	vSqlUpdateFieldLogOffs = "LogWLogOffset" //nolint ST1003
	vSqlUpdateFieldCUDOffs = "CUDWLogOffset" //nolint ST1003
)

var (
	qNameCmdVSqlUpdate     = appdef.NewQName("cluster", "VSqlUpdate")
	qNameQryVSqlUpdate2    = appdef.NewQName("cluster", "VSqlUpdate2")
	resourceCmdVSqlUpdate  = "c." + qNameCmdVSqlUpdate.String()
	resourceQryVSqlUpdate2 = "q." + qNameQryVSqlUpdate2.String()
)

func isVSqlUpdateV1Call(busRequest bus.Request) bool {
	return !busRequest.IsAPIV2 && busRequest.Resource == resourceCmdVSqlUpdate && isUpdateTableBody(busRequest.Body)
}

func isVSqlUpdateV2Call(busRequest bus.Request) bool {
	return busRequest.IsAPIV2 &&
		processors.APIPath(busRequest.APIPath) == processors.APIPath_Commands &&
		busRequest.QName == qNameCmdVSqlUpdate &&
		isUpdateTableBody(busRequest.Body)
}

// isUpdateTableBody returns true if the VSqlUpdate body carries an "update table" DML.
// Only this kind is routed through the query shim; any other kind stays on the command path.
// On any error the request will be forwarded to processor where the error will be actually handled, so do not handle errors here
func isUpdateTableBody(body []byte) bool {
	m := map[string]any{}
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	args, ok := m["args"].(map[string]any)
	if !ok {
		return false
	}
	q, ok := args["Query"].(string)
	if !ok {
		return false
	}
	op, err := dml.ParseQuery(q)
	if err != nil {
		return false
	}
	return op.Kind == dml.OpKind_UpdateTable
}

func rewriteVSqlUpdateBody(body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"elements":[{"fields":["`)
	buf.WriteString(vSqlUpdateFieldLogOffs)
	buf.WriteString(`","`)
	buf.WriteString(vSqlUpdateFieldCUDOffs)
	buf.WriteString(`"]}]`)
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 2 {
		buf.WriteByte(',')
		buf.Write(trimmed[1 : len(trimmed)-1])
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

// capturingResponseWriter buffers the body and captures headers/status so reply_v1 and
// reply_v2 can run unchanged while the shim rewrites the final response shape.
type capturingResponseWriter struct {
	headers http.Header
	body    bytes.Buffer
	status  int
}

func newCapturingResponseWriter() *capturingResponseWriter {
	return &capturingResponseWriter{headers: http.Header{}, status: http.StatusOK}
}

func (c *capturingResponseWriter) Header() http.Header         { return c.headers }
func (c *capturingResponseWriter) Write(b []byte) (int, error) { return c.body.Write(b) }
func (c *capturingResponseWriter) WriteHeader(code int)        { c.status = code }
func (c *capturingResponseWriter) Flush()                      {}

func (c *capturingResponseWriter) flushTo(rw http.ResponseWriter, overrideBody string) {
	for k, v := range c.headers {
		rw.Header()[k] = v
	}
	rw.WriteHeader(c.status)
	if len(overrideBody) > 0 {
		writeResponse(rw, overrideBody)
		return
	}
	writeResponse(rw, c.body.String())
}

func dispatchVSqlUpdateShim_V1(requestCtx context.Context, rw http.ResponseWriter, busRequest bus.Request, reqSender bus.IRequestSender) {
	busRequest.Resource = resourceQryVSqlUpdate2
	busRequest.Body = rewriteVSqlUpdateBody(busRequest.Body)

	respCh, respMeta, respErr, err := reqSender.SendRequest(requestCtx, busRequest)
	if err != nil {
		logger.ErrorCtx(requestCtx, "routing.send2vvm.error", "forwarding c.cluster.VSqlUpdate to q.cluster.VSqlUpdate2 failed:", err)
		writeCommonError_V1(rw, err, http.StatusInternalServerError)
		return
	}

	capture := newCapturingResponseWriter()
	initResponse(capture, respMeta)
	reply_v1(requestCtx, capture, respCh, respErr, func() {}, busRequest, respMeta)

	overrideBody := ""
	if capture.status == http.StatusOK && *respErr == nil {
		overrideBody = fmt.Sprintf(`{"CurrentWLogOffset":%d}`, extractLogWLogOffsetFromV1Body(capture.body.Bytes()))
	}
	capture.flushTo(rw, overrideBody)
}

func dispatchVSqlUpdateShim_V2(requestCtx context.Context, rw http.ResponseWriter, busRequest bus.Request, reqSender bus.IRequestSender) {
	args := map[string]any{}
	if len(busRequest.Body) > 0 {
		body := map[string]any{}
		if err := json.Unmarshal(busRequest.Body, &body); err != nil {
			ReplyCommonError(rw, fmt.Sprintf("failed to parse VSqlUpdate body: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if a, ok := body["args"].(map[string]any); ok {
			args = a
		}
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		// notest
		ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if busRequest.Query == nil {
		busRequest.Query = map[string]string{}
	}
	busRequest.Query["args"] = string(argsBytes)
	busRequest.Query["keys"] = vSqlUpdateFieldLogOffs + "," + vSqlUpdateFieldCUDOffs
	busRequest.Method = http.MethodGet
	busRequest.APIPath = int(processors.APIPath_Queries)
	busRequest.QName = qNameQryVSqlUpdate2
	busRequest.Resource = resourceQryVSqlUpdate2
	busRequest.Body = nil

	respCh, respMeta, respErr, err := reqSender.SendRequest(requestCtx, busRequest)
	if err != nil {
		logger.ErrorCtx(requestCtx, "routing.send2vvm.error", "forwarding cluster.VSqlUpdate to cluster.VSqlUpdate2 failed:", err)
		ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	capture := newCapturingResponseWriter()
	initResponse(capture, respMeta)
	reply_v2(requestCtx, capture, respCh, respErr, func() {}, respMeta)

	overrideBody := ""
	if capture.status == http.StatusOK && *respErr == nil {
		overrideBody = fmt.Sprintf(`{"currentWLogOffset":%d}`, extractLogWLogOffsetFromV2Body(capture.body.Bytes()))
	}
	capture.flushTo(rw, overrideBody)
}

// extractLogWLogOffsetFromV1Body parses the v1 query response envelope
// `{"sections":[{"type":"","elements":[[[[LogWLogOffset, CUDWLogOffset]]]]}]}`
// and returns the first LogWLogOffset value.
func extractLogWLogOffsetFromV1Body(raw []byte) int64 {
	var env federation.QueryResponse
	if err := json.Unmarshal(raw, &env); err != nil {
		// notest
		return 0
	}
	if v, ok := env.Sections[0].Elements[0][0][0][0].(float64); ok {
		return int64(v)
	}
	// notest
	return 0
}

// extractLogWLogOffsetFromV2Body parses the v2 query response envelope
// `{"results":[{"LogWLogOffset":..., "CUDWLogOffset":...}]}` and returns the first
// LogWLogOffset value.
func extractLogWLogOffsetFromV2Body(raw []byte) int64 {
	var env federation.FuncResponse
	if err := json.Unmarshal(raw, &env); err != nil {
		// notest
		return 0
	}
	if len(env.QPv2Response) == 0 {
		// notest
		return 0
	}
	if v, ok := env.QPv2Response[0][vSqlUpdateFieldLogOffs].(float64); ok {
		return int64(v)
	}
	// notest
	return 0
}
