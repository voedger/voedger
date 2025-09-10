/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestIsBlank(t *testing.T) {
	require := require.New(t)
	cases := []struct {
		in  string
		out bool
	}{
		{"", true},
		{" ", true},
		{"\n\t", true},
		{"a", false},
		{" a ", false},
	}
	for idx := range cases {
		require.Equal(cases[idx].out, IsBlank(cases[idx].in))
	}
}

func TestIsDebug(t *testing.T) {
	withArgs([]string{"/tmp/__debug_bin"}, func() {
		require.True(t, IsDebug())
	})
	withArgs([]string{"/tmp/normal_bin"}, func() {
		require.False(t, IsDebug())
	})
}

func TestIsCassandraStorage(t *testing.T) {
	t.Setenv("CASSANDRA_TESTS_ENABLED", "1")
	require.True(t, IsCassandraStorage())
}

func TestIsDynamoDBStorage(t *testing.T) {
	t.Setenv("DYNAMODB_TESTS_ENABLED", "1")
	require.True(t, IsDynamoDBStorage())
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

type errUnwrapper struct{ errs []error }

func (e errUnwrapper) Error() string   { return "wrapped" }
func (e errUnwrapper) Unwrap() []error { return e.errs }

func TestSplitErrors(t *testing.T) {
	require := require.New(t)
	err := errors.New("err1")
	require.Equal([]error{err}, SplitErrors(err))
	require.Nil(SplitErrors(nil))
	wrapped := errUnwrapper{[]error{err, err}}
	require.Equal([]error{err, err}, SplitErrors(wrapped))
}

func TestNilAdminPortGetter(t *testing.T) {
	require := require.New(t)
	require.Panics(func() { NilAdminPortGetter() })
}

func TestScanSSE(t *testing.T) {
	cases := []struct {
		name  string
		data  []byte
		atEOF bool
		adv   int
		token []byte
	}{
		{"double_newline", []byte("event: x\n\ndata: y\n\n"), false, 10, []byte("event: x")},
		{"empty_atEOF", []byte{}, true, 0, nil},
		{"no_newline_atEOF", []byte("abc"), true, 3, []byte("abc")},
		{"no_newline_notEOF", []byte("abc"), false, 0, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			advance, token, err := ScanSSE(c.data, c.atEOF)
			require.NoError(t, err)
			require.Equal(t, c.adv, advance)
			require.Equal(t, c.token, token)
		})
	}
}

func TestInt64ToWSID(t *testing.T) {
	require := require.New(t)
	ok, err := Int64ToWSID(1)
	require.NoError(err)
	require.Equal(istructs.WSID(1), ok)
	_, err = Int64ToWSID(-1)
	require.Error(err)
	_, err = Int64ToWSID(int64(istructs.MaxAllowedWSID))
	require.NoError(err)
}

func TestInt64ToRecordID(t *testing.T) {
	require := require.New(t)
	ok, err := Int64ToRecordID(1)
	require.NoError(err)
	require.Equal(istructs.RecordID(1), ok)
	_, err = Int64ToRecordID(-1)
	require.Error(err)
}

func withArgs(args []string, f func()) {
	orig := make([]string, len(os.Args))
	copy(orig, os.Args)
	defer func() { os.Args = orig }()
	os.Args = args
	f()
}
