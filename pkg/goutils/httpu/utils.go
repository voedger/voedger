/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"
)

func readBody(resp *http.Response) (string, error) {
	respBody, err := io.ReadAll(resp.Body)
	return string(respBody), err
}

func discardRespBody(resp *http.Response) error {
	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		// https://github.com/voedger/voedger/issues/1694
		if !IsWSAEError(err, WSAECONNRESET) {
			return fmt.Errorf("failed to discard response body: %w", err)
		}
	}
	return nil
}

func DenyGETAndDiscardResponse(opts IReqOpts) (panicMessage string) {
	if opts.httpOpts().discardResp && opts.httpOpts().method == http.MethodGet {
		return "WithDiscardResponse is denied on GET method"
	}
	return ""
}

func IsWSAEError(err error, errno syscall.Errno) bool {
	var sysCallErr *os.SyscallError
	if errors.As(err, &sysCallErr) {
		var syscallErrno syscall.Errno
		if errors.As(sysCallErr.Err, &syscallErrno) {
			return syscallErrno == errno
		}
	}
	return false
}
