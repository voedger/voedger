/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAndFilter_IsMatch(t *testing.T) {
	t.Run("Truth table", func(t *testing.T) {
		match := func(match bool, err error) bool {
			require.NoError(t, err)
			return match
		}
		template := "First %t second %t want %t"
		tests := []struct {
			first  bool
			second bool
			want   bool
		}{
			{
				first:  false,
				second: false,
				want:   false,
			},
			{
				first:  true,
				second: false,
				want:   false,
			},
			{
				first:  false,
				second: true,
				want:   false,
			},
			{
				first:  true,
				second: true,
				want:   true,
			},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf(template, test.first, test.second, test.want), func(t *testing.T) {
				filter := AndFilter{
					filters: []IFilter{
						testFilter{match: test.first},
						testFilter{match: test.second},
					},
				}

				require.Equal(t, test.want, match(filter.IsMatch(nil, nil)))
			})
		}
	})
	t.Run("Should return error", func(t *testing.T) {
		require := require.New(t)
		filter := AndFilter{
			filters: []IFilter{
				testFilter{
					match: false,
					err:   ErrWrongType,
				},
			},
		}

		match, err := filter.IsMatch(nil, nil)

		require.ErrorIs(err, ErrWrongType)
		require.False(match)
	})
}
