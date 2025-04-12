/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils"
)

var onBeforeWriteResponse func(w http.ResponseWriter) // not nil in tests only

func WriteTextResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(coreutils.ContentType, "text/plain")
	w.WriteHeader(code)
	writeResponse(w, msg)
}

func WriteJSONResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
	w.WriteHeader(code)

	response := map[string]string{"message": msg}
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal JSON response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeResponse(w, `{"message": "Internal Server Error"}`)
		return
	}
	writeResponse(w, string(jsonData))
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

func writeNotImplemented(rw http.ResponseWriter) {
	WriteTextResponse(rw, "not implemented", http.StatusNotImplemented)
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
