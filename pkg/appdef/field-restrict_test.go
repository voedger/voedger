/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinLen(t *testing.T) {
	require := require.New(t)

	require.NotNil(MinLen(1), "must be ok to obtain MinLen")

	require.Panics(func() { _ = MinLen(MaxStringFieldLength + 1) }, "must be panic to obtain MinLen with too large argument")
}

func TestMaxLen(t *testing.T) {
	require := require.New(t)

	require.NotNil(MaxLen(1), "must be ok to obtain MaxLen")

	require.Panics(func() { _ = MaxLen(0) }, "must be panic to obtain MaxLen with zero argument")
	require.Panics(func() { _ = MaxLen(MaxStringFieldLength + 1) }, "must be panic to obtain MaxLen with too large argument")
}

func TestPattern(t *testing.T) {
	require := require.New(t)

	require.NotNil(Pattern(`^/a+$`), "must be ok to obtain Pattern")

	require.Panics(func() { _ = Pattern(`(]`) }, "must be panic to obtain Pattern with invalid Regexp")
}

func Test_fieldRestricts(t *testing.T) {
	require := require.New(t)

	t.Run("check empty restrict members", func(t *testing.T) {
		empty := newFieldRestricts()
		require.Zero(empty.MinLen())
		require.EqualValues(DefaultStringFieldMaxLength, empty.MaxLen())
		require.Nil(empty.Pattern())
	})

	t.Run("check restrict members", func(t *testing.T) {
		r := newFieldRestricts(
			MinLen(1),
			MaxLen(4),
			Pattern(`^/a+$`),
		)
		require.EqualValues(1, r.MinLen())
		require.EqualValues(4, r.MaxLen())
		require.EqualValues(`^/a+$`, r.Pattern().String())
	})

	require.Panics(func() { _ = newFieldRestricts(MinLen(2), MaxLen(1)) }, "must be panic is incompatible restricts")
}
