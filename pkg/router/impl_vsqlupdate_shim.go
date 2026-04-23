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
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/processors"
)

const (
	vSqlUpdateFieldLogOffs = "LogWLogOffset" //nolint ST1003
	vSqlUpdateFieldCUDOffs = "CUDWLogOffset" //nolint ST1003
	vSqlUpdateFieldNewID   = "NewID"         //nolint ST1003
	vsqlUpdateStage        = "routing.vsqlupdate"
	vsqlUpdateErrorStage   = "routing.vsqlupdate.error"
)

var (
	qNameCmdVSqlUpdate     = appdef.NewQName("cluster", "VSqlUpdate")
	qNameQryVSqlUpdate2    = appdef.NewQName("cluster", "VSqlUpdate2")
	resourceCmdVSqlUpdate  = "c." + qNameCmdVSqlUpdate.String()
	resourceQryVSqlUpdate2 = "q." + qNameQryVSqlUpdate2.String()
)

func isVSqlUpdateV1Call(busRequest bus.Request) bool {
	return !busRequest.IsAPIV2 && busRequest.Resource == resourceCmdVSqlUpdate
}

func isVSqlUpdateV2Call(busRequest bus.Request) bool {
	return busRequest.IsAPIV2 &&
		processors.APIPath(busRequest.APIPath) == processors.APIPath_Commands &&
		busRequest.QName == qNameCmdVSqlUpdate
}

func rewriteVSqlUpdateBody(body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"elements":[{"fields":["`)
	buf.WriteString(vSqlUpdateFieldLogOffs)
	buf.WriteString(`","`)
	buf.WriteString(vSqlUpdateFieldCUDOffs)
	buf.WriteString(`","`)
	buf.WriteString(vSqlUpdateFieldNewID)
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
	logger.VerboseCtx(requestCtx, vsqlUpdateStage, fmt.Sprintf("rerouting %s to %s", resourceCmdVSqlUpdate, resourceQryVSqlUpdate2))

	busRequest.Resource = resourceQryVSqlUpdate2
	busRequest.Body = rewriteVSqlUpdateBody(busRequest.Body)

	respCh, respMeta, respErr, err := reqSender.SendRequest(requestCtx, busRequest)
	if err != nil {
		logger.ErrorCtx(requestCtx, "routing.send2vvm.error", fmt.Sprintf("forwarding %s to %s failed: %s", resourceCmdVSqlUpdate, resourceQryVSqlUpdate2, err))
		writeCommonError_V1(rw, err, http.StatusInternalServerError)
		return
	}

	capture := newCapturingResponseWriter()
	initResponse(capture, respMeta)
	reply_v1(requestCtx, capture, respCh, respErr, func() {}, busRequest, respMeta)

	finalizeShimResponse(requestCtx, rw, capture, respErr, extractFromQryVSqlUpdate2ResponseV1, "CurrentWLogOffset", "Result")
}

func dispatchVSqlUpdateShim_V2(requestCtx context.Context, rw http.ResponseWriter, busRequest bus.Request, reqSender bus.IRequestSender) bool {
	args := map[string]any{}
	if len(busRequest.Body) > 0 {
		body := map[string]any{}
		if err := json.Unmarshal(busRequest.Body, &body); err != nil {
			return false
		}
		if a, ok := body["args"].(map[string]any); ok {
			args = a
		} else {
			return false
		}
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return false
	}

	logger.VerboseCtx(requestCtx, vsqlUpdateStage, fmt.Sprintf("rerouting %s to %s", resourceCmdVSqlUpdate, resourceQryVSqlUpdate2))

	if busRequest.Query == nil {
		busRequest.Query = map[string]string{}
	}
	busRequest.Query["args"] = string(argsBytes)
	busRequest.Query["keys"] = vSqlUpdateFieldLogOffs + "," + vSqlUpdateFieldCUDOffs + "," + vSqlUpdateFieldNewID
	busRequest.Method = http.MethodGet
	busRequest.APIPath = int(processors.APIPath_Queries)
	busRequest.QName = qNameQryVSqlUpdate2
	busRequest.Resource = resourceQryVSqlUpdate2
	busRequest.Body = nil

	respCh, respMeta, respErr, err := reqSender.SendRequest(requestCtx, busRequest)
	if err != nil {
		return false
	}

	capture := newCapturingResponseWriter()
	initResponse(capture, respMeta)
	reply_v2(requestCtx, capture, respCh, respErr, func() {}, respMeta)

	finalizeShimResponse(requestCtx, rw, capture, respErr, extractFromQryVSqlUpdate2ResponseV2, "currentWLogOffset", "result")
	return true
}

func finalizeShimResponse(requestCtx context.Context, rw http.ResponseWriter, capture *capturingResponseWriter, respErr *error,
	extract func([]byte) (logOffset, cudOffset int64, newID int64), offsetKey, resultKey string) {
	overrideBody := ""
	if capture.status == http.StatusOK && *respErr == nil {
		logOffset, cudOffset, newID := extract(capture.body.Bytes())
		logger.VerboseCtx(requestCtx, vsqlUpdateStage, fmt.Sprintf("%s=%d (to be sent to the client as CurrentWLogOffset), %s=%d", vSqlUpdateFieldLogOffs, logOffset, vSqlUpdateFieldCUDOffs, cudOffset))
		overrideBody = buildCmdResponse(logOffset, newID, offsetKey, resultKey)
	} else {
		logger.ErrorCtx(requestCtx, vsqlUpdateErrorStage, fmt.Sprintf("%s shim reply failed: status=%d respErr=%v body=%s", resourceCmdVSqlUpdate, capture.status, *respErr, capture.body.String()))
	}
	capture.flushTo(rw, overrideBody)
}

func buildCmdResponse(logOffset int64, newID int64, offsetKey, resultKey string) string {
	if newID > 0 {
		return fmt.Sprintf(`{%q:%d,%q:{"NewID":%d}}`, offsetKey, logOffset, resultKey, newID)
	}
	return fmt.Sprintf(`{%q:%d}`, offsetKey, logOffset)
}

// extractFromQryVSqlUpdate2ResponseV1 parses the v1 query response envelope
// `{"sections":[{"type":"","elements":[[[[LogWLogOffset, CUDWLogOffset, NewID]]]]}]}`
// and returns LogWLogOffset, CUDWLogOffset and NewID.
func extractFromQryVSqlUpdate2ResponseV1(raw []byte) (logOffset int64, cudOffset int64, newID int64) {
	var env federation.QueryResponse
	if err := json.Unmarshal(raw, &env); err != nil {
		// notest
		panic(err)
	}

	// following data is got without any checking since we know for sure the data structure of the query result
	// guarded by integration tests
	row := env.Sections[0].Elements[0][0][0]
	logOffset = int64(row[0].(float64))
	cudOffset = int64(row[1].(float64))
	newID = int64(row[2].(float64))

	return logOffset, cudOffset, newID
}

// extractFromQryVSqlUpdate2ResponseV2 parses the v2 query response envelope
// `{"results":[{"LogWLogOffset":..., "CUDWLogOffset":..., "NewID":...}]}` and returns
// LogWLogOffset, CUDWLogOffset and NewID.
func extractFromQryVSqlUpdate2ResponseV2(raw []byte) (logOffset int64, cudOffset int64, newID int64) {
	var env federation.FuncResponse
	if err := json.Unmarshal(raw, &env); err != nil {
		// notest
		panic(err)
	}

	// following data is got without any checking since we know for sure the data structure of the query result
	// guarded by integration tests
	logOffset = int64(env.QPv2Response[0][vSqlUpdateFieldLogOffs].(float64))
	cudOffset = int64(env.QPv2Response[0][vSqlUpdateFieldCUDOffs].(float64))
	newID = int64(env.QPv2Response[0][vSqlUpdateFieldNewID].(float64))

	return logOffset, cudOffset, newID
}
