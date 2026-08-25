/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestReadBodyClosesResponseBody(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader("body")}
		got, err := readBody(&http.Response{Body: body})
		require.NoError(t, err)
		require.Equal(t, "body", got)
		require.True(t, body.closed)
	})

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("read failed")
		body := &closeTrackingBody{Reader: failingReader{err: readErr}}
		got, err := readBody(&http.Response{Body: body})
		require.ErrorIs(t, err, readErr)
		require.Empty(t, got)
		require.True(t, body.closed)
	})
}

func TestIsWSAEError(t *testing.T) {
	require := require.New(t)
	err := &os.SyscallError{Err: syscall.Errno(123)}
	require.True(IsWSAEError(err, 123))
	require.False(IsWSAEError(err, 124))
	require.False(IsWSAEError(errors.New("x"), 123))
}

func TestServerAddress(t *testing.T) {
	require := require.New(t)

	t.Run("LocalhostAddress binds to localhost:0 only", func(t *testing.T) {
		require.Equal("127.0.0.1:0", LocalhostDynamic())
	})

	t.Run("ListenAddr returns localhost:0 for port 0", func(t *testing.T) {
		require.Equal("127.0.0.1:0", ListenAddr(0))
	})

	t.Run("ListenAddr returns public address for non-zero port", func(t *testing.T) {
		require.Equal(":8080", ListenAddr(8080))
	})
}

func TestParseRetryAfterHeader(t *testing.T) {
	t.Run("seconds format", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"120"},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, 120*time.Second, duration)
	})

	t.Run("HTTP date format", func(t *testing.T) {
		futureTime := time.Now().Add(5 * time.Minute)
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{futureTime.UTC().Format(http.TimeFormat)},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Greater(t, duration, 4*time.Minute)
		require.Less(t, duration, 6*time.Minute)
	})

	t.Run("past HTTP date", func(t *testing.T) {
		pastTime := time.Now().Add(-5 * time.Minute)
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{pastTime.UTC().Format(http.TimeFormat)},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, time.Duration(0), duration)
	})

	t.Run("invalid format", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"invalid"},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, time.Duration(0), duration)
	})

	t.Run("missing header", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, time.Duration(0), duration)
	})

	t.Run("zero seconds", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"0"},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, time.Duration(0), duration)
	})

	t.Run("negative seconds", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"-10"},
			},
		}
		duration := parseRetryAfterHeader(resp)
		require.Equal(t, time.Duration(0), duration)
	})
}
