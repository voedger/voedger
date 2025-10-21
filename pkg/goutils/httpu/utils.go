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
	"strconv"
	"syscall"
	"time"
)

func readBody(resp *http.Response) (string, error) {
	respBody, err := io.ReadAll(resp.Body)
	return string(respBody), err
}

func discardRespBody(resp *http.Response) error {
	defer resp.Body.Close()
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

func ListenAddr(port int) string {
	if port == 0 {
		return LocalhostDynamic()
	}
	return fmt.Sprintf(":%d", port)
}

// LocalhostDynamic returns a server address that binds only to localhost (127.0.0.1:0)
// Use this for tests, admin interfaces, or services that should only be locally accessible
func LocalhostDynamic() string {
	return fmt.Sprintf("%s:0", LocalhostIP.String())
}

// parseRetryAfterHeader parses the Retry-After header value.
// It supports both seconds format (e.g., "120") and HTTP date format (e.g., "Wed, 21 Oct 2015 07:28:00 GMT").
// Returns the duration to wait, or 0 if the header is invalid or not present.
func parseRetryAfterHeader(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get(RetryAfter)
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if retryTime, err := http.ParseTime(retryAfter); err == nil {
		// according to HTTP Standard (RFC 7231) the time must be in UTC
		nowUTC := time.Now().UTC()
		duration := retryTime.Sub(nowUTC)
		if duration > 0 {
			return duration
		}
	}

	// Invalid or past date
	return 0
}
