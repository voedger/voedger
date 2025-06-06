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
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils"
)

var onBeforeWriteResponse func(w http.ResponseWriter) // not nil in tests only

func WriteTextResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(coreutils.ContentType, "text/plain")
	w.WriteHeader(code)
	writeResponse(w, msg)
}

func ReplyCommonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
	w.WriteHeader(code)
	writeCommonError(w, msg, code)
}

func ReplyJSON(w http.ResponseWriter, data string, code int) {
	w.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
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
