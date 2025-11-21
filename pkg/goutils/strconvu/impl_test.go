/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package strconvu

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestParseUint8_edgeCases(t *testing.T) {
	require := require.New(t)
	// Edge cases
	t.Run("ok", func(t *testing.T) {
		cases := []struct {
			str      string
			expected uint8
		}{
			{"0", 0},
			{"255", 255},
			{"00", 0},
			{"007", 7},
		}
		for _, c := range cases {
			actual, err := ParseUint8(c.str)
			require.NoError(err)
			require.Equal(c.expected, actual)
		}
	})
	t.Run("bad", func(t *testing.T) {
		cases := []string{
			" 42 ",
			"42.0",
			"",
		}
		for _, c := range cases {
			actual, err := ParseUint8(c)
			require.Error(err)
			require.Zero(actual)
		}
	})
}

func TestParseInt32_edgeCases(t *testing.T) {
	require := require.New(t)
	t.Run("ok", func(t *testing.T) {
		cases := []struct {
			str      string
			expected int32
		}{
			{"0", 0},
			{"2147483647", 2147483647},
			{"-2147483648", -2147483648},
			{"007", 7},
		}
		for _, c := range cases {
			actual, err := ParseInt32(c.str)
			require.NoError(err)
			require.Equal(c.expected, actual)
		}
	})
	t.Run("bad", func(t *testing.T) {
		cases := []string{
			" 42 ",
			"42.0",
			"",
			"2147483648",  // overflow above max int32
			"-2147483649", // underflow below min int32
		}
		for _, c := range cases {
			actual, err := ParseInt32(c)
			require.Error(err)
			require.Zero(actual)
		}
	})
}
