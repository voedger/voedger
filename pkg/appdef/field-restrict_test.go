/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_fieldRestrict_String(t *testing.T) {
	tests := []struct {
		name string
		r    fieldRestrict
		want string
	}{
		{"MinLen -> MinLen", fieldRestrict_MinLen, "MinLen"},
		{"fieldRestrict_Count -> fieldRestrict(3)", fieldRestrict_Count, fmt.Sprintf("fieldRestrict(%d)", fieldRestrict_Count)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmt.Sprintf("%v", tt.r); got != tt.want {
				t.Errorf("fieldRestrict.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinLen(t *testing.T) {
	require := require.New(t)

	require.NotNil(MinLen(1), "must be ok to obtain MinLen(1+)")

	require.Panics(func() { _ = MinLen(MaxFieldLength + 1) }, "must be panic to obtain MinLen(tooLarge)")
}

func TestMaxLen(t *testing.T) {
	require := require.New(t)

	require.NotNil(MaxLen(1), "must be ok to obtain MaxLen(1+)")

	require.Panics(func() { _ = MaxLen(0) }, "must be panic to obtain MaxLen(0)")
	require.Panics(func() { _ = MaxLen(MaxFieldLength + 1) }, "must be panic to obtain MaxLen(tooLarge)")
}

func TestPattern(t *testing.T) {
	require := require.New(t)

	require.NotNil(Pattern(`^/a+$`), "must be ok to obtain Pattern")

	require.Panics(func() { _ = Pattern(`(]`) }, "must be panic to obtain Pattern with invalid Regexp")
}

func Test_fieldRestricts(t *testing.T) {
	require := require.New(t)

	t.Run("check empty restrict members", func(t *testing.T) {
		empty := newCharsField("test", DataKind_string, false).Restricts()
		require.Zero(empty.MinLen())
		require.EqualValues(DefaultFieldMaxLength, empty.MaxLen())
		require.Nil(empty.Pattern())
	})

	t.Run("check restrict members", func(t *testing.T) {
		r := newCharsField("test", DataKind_string, false,
			MinLen(1),
			MaxLen(4),
			Pattern(`^/a+$`),
		).Restricts()
		require.EqualValues(1, r.MinLen())
		require.EqualValues(4, r.MaxLen())
		require.EqualValues(`^/a+$`, r.Pattern().String())
	})

	require.Panics(func() { _ = newCharsField("test", DataKind_bytes, false, MinLen(2), MaxLen(1)) }, "must be panic is incompatible restricts")
}

func Test_fieldRestricts_String(t *testing.T) {
	tests := []struct {
		name  string
		field *charsField
		want  string
	}{
		{`empty -> ""`, newCharsField("test", DataKind_bytes, false), ""},
		{`MinLen(4) -> "MinLen: 4"`,
			newCharsField("test", DataKind_bytes, false, MinLen(4)),
			"MinLen: 4"},
		{`MinLen(4), MaxLen(10) -> "MinLen: 4, MaxLen: 10"`,
			newCharsField("test", DataKind_bytes, false, MinLen(4), MaxLen(10)),
			"MinLen: 4, MaxLen: 10"},
		{`MinLen(4), MaxLen(10), Pattern('^\d+$') -> "MinLen: 4, MaxLen: 10, Pattern: '^\d+$'"`,
			newCharsField("test", DataKind_string, false, MinLen(4), MaxLen(10), Pattern(`^\d+$`)),
			"MinLen: 4, MaxLen: 10, Pattern: `^\\d+$`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmt.Sprintf("%v", tt.field.Restricts()); got != tt.want {
				t.Errorf("fieldRestricts.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
