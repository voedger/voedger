/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/strconvu"
	"github.com/voedger/voedger/pkg/istructs"
)

var onBeforeWriteResponse func(w http.ResponseWriter) // not nil in tests only

func WriteTextResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(httpu.ContentType, "text/plain")
	w.WriteHeader(code)
	writeResponse(w, msg)
}

func ReplyCommonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(httpu.ContentType, httpu.ContentType_ApplicationJSON)
	w.WriteHeader(code)
	writeCommonError(w, msg, code)
}

func ReplyJSON(w http.ResponseWriter, data string, code int) {
	w.Header().Set(httpu.ContentType, httpu.ContentType_ApplicationJSON)
	w.WriteHeader(code)
	writeResponse(w, data)
}

func writeCommonError(w http.ResponseWriter, msg string, code int) bool {
	return writeResponse(w, fmt.Sprintf(`{"status":%d,"message":%q}`, code, msg))
}

func writeResponse(w http.ResponseWriter, data string) bool {
	if onBeforeWriteResponse != nil {
		onBeforeWriteResponse(w)
	}
	if _, err := w.Write([]byte(data)); err != nil {
		stack := debug.Stack()
		log.Println("failed to write response:", err, "\n", string(stack))
		return false
	}
	w.(http.Flusher).Flush()
	return true
}

type filteringWriter struct {
	w io.Writer
}

func (fw *filteringWriter) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), "TLS handshake error") {
		return len(p), nil
	}
	return fw.w.Write(p)
}

func replyServiceUnavailable(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusServiceUnavailable)
	rw.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
}

func replyErr(rw http.ResponseWriter, err error) {
	var sysError coreutils.SysError
	if errors.As(err, &sysError) {
		ReplyJSON(rw, sysError.ToJSON_APIV2(), sysError.HTTPStatus)
	} else {
		ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
	}
}

// copies Authorization cookie (if present) to header (if missing)
// needed for blobs and n10n.
func GetCookieBearerAuth(req *http.Request) (cookieBearerToken string, ok bool, err error) {
	cookie, err := req.Cookie(httpu.Authorization)
	if errors.Is(err, http.ErrNoCookie) {
		return "", false, nil
	}
	if err != nil {
		// notest
		return "", false, fmt.Errorf("failed to read cookie: %w", err)
	}
	if cookieBearerToken, err = url.QueryUnescape(cookie.Value); err != nil {
		return "", false, fmt.Errorf("failed to unescape cookie value %q: %w", cookie.Value, err)
	}
	return cookieBearerToken, true, nil
}

// createBusRequest creates a bus.Request from validated data
func createBusRequest(reqMethod string, data validatedData, req *http.Request) bus.Request {
	res := bus.Request{
		Method:   reqMethod,
		WSID:     data.wsid,
		Query:    map[string]string{},
		Header:   data.header,
		AppQName: data.appQName,
		Resource: data.vars[URLPlaceholder_resourceName],
		Body:     data.body,
	}

	if docIDStr, hasDocID := data.vars[URLPlaceholder_id]; hasDocID {
		docIDUint64, err := strconvu.ParseUint64(docIDStr)
		if err != nil {
			// notest: parsed already by route regexp
			panic(err)
		}
		res.DocID = istructs.IDType(docIDUint64)
	}

	for k, v := range req.URL.Query() {
		res.Query[k] = v[0]
	}
	return res
}
