/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istorage

import (
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/pkg/istructs"
)

func TestGetAppStorgaDesc(t *testing.T) {
	require := require.New(t)
	type n struct {
		name     istructs.AppQName
		expected string
	}
	cases := [][]n{
		{
			// basic
			{istructs.NewAppQName("", ""), "{uuid}"},
			{istructs.NewAppQName("sys", "ok"), "sysok"},
			{istructs.NewAppQName("sys", "12ok "), "sys12ok"},
			{istructs.NewAppQName("sys", "12OK "), "sys12ok{uuid}"},
			{istructs.NewAppQName("sys", "12OK_"), "sys12ok{uuid}"},
			{istructs.NewAppQName("sys", "_"), "sys"},
			{istructs.NewAppQName("sys", "a"), "sysa"},
			{istructs.NewAppQName("sys", "!"), "sys{uuid}"},
		},
		{
			// first char must not be a digit
			{istructs.NewAppQName("1sys", "_"), "asys"},
			{istructs.NewAppQName("2sys", "_"), "asys{uuid}"},
			{istructs.NewAppQName("1sys", "6_"), "asys6"},
			{istructs.NewAppQName("1sys", "ok"), "asysok"},
			{istructs.NewAppQName("11sys", "ok"), "a1sysok"},
			{istructs.NewAppQName("21sys", "ok"), "a1sysok{uuid}"},
		},
		{
			// matches caused by triming differences
			// no trim                                                      43 chars                                             43 chars
			{istructs.NewAppQName("sys", "aaaaaaaaAaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "sysaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},

			// +1 symbol -> trim, last symbol replaced                      44 chars                 43 chars
			{istructs.NewAppQName("sys", "aaaaaaaaaaaaaaaaaaaaaaAaaaaaaaaaaaaaaaaaa"), "sysaaaaaaa{uuid}"},
			{istructs.NewAppQName("sys", "aaaaaaaaaaaaaaAaaaaaaaaaaaaaaaaaaaaaaaaab"), "sysaaaaaaa{uuid}"},

			// +1 symbol -> trim, last symbol replaced + match with previous *b
			{istructs.NewAppQName("sys", "aaaaaaaaaaaaaaaAaaaaaaaaaaaaaaaaaaaaaaaaaaaaba"), "sysaaaaaaaaaaaaa{uuid}"},

			// all wrong chars are replaced with good ones, already have a match
			{istructs.NewAppQName("sys", `bcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRS`), "sysbcdefghijklmnopqrstuvwxyzabcdefghijklmno"},
			{istructs.NewAppQName("sys", `_____________________________________________`), "sys"},
		},
		{
			// non-latin chars
			{istructs.NewAppQName("sys", `Карлосон 哇"呀呀`), "sys"},
			{istructs.NewAppQName("sys", `aaaaaaaaaaaaa`), "sysaaaaaaaaaaaaa"},
		},
	}
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{27}$`)
	for _, c := range cases {
		names := map[string]bool{}
		for _, n := range c {
			san, err := NewSafeAppName(n.name, func(name string) (bool, error) {
				return !names[name], nil
			})
			require.NoError(err)
			names[san.String()] = true
			if uuidPlaceholderPos := strings.Index(n.expected, "{uuid}"); uuidPlaceholderPos >= 0 {
				uuidProbe := san.String()[uuidPlaceholderPos : uuidPlaceholderPos+27]
				require.Regexp(uuidRegex, uuidProbe, n.name)
			} else {
				require.Equal(n.expected, san.String(), n.name)
			}
			log.Printf("%s -> %s", n.name, san)
		}
	}
}

func TestSafeAppNameJSON(t *testing.T) {
	require := require.New(t)

	t.Run("value", func(t *testing.T) {
		expected := SafeAppName{name: `sysmy`}
		b, err := json.Marshal(&expected)
		require.NoError(err)

		var actual SafeAppName
		err = json.Unmarshal(b, &actual)
		require.NoError(err)

		require.Equal(expected, actual)
		log.Println(string(b))
	})

	t.Run("pointer", func(t *testing.T) {
		expected := &SafeAppName{name: `sysmy`}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		var actual *SafeAppName
		err = json.Unmarshal(b, &actual)
		require.NoError(err)

		require.Equal(expected, actual)
		log.Println(string(b))
	})

	t.Run("key of a map", func(t *testing.T) {
		name := SafeAppName{name: `sysmy`}
		expected := map[SafeAppName]bool{
			name: true,
		}

		b, err := json.Marshal(&expected)
		require.NoError(err)

		var actual map[SafeAppName]bool
		err = json.Unmarshal(b, &actual)
		require.NoError(err)

		require.Equal(expected, actual)
		log.Println(string(b))
	})
}

func TestNewSafeAppNameErrors(t *testing.T) {
	require := require.New(t)
	t.Run("no safe app name", func(t *testing.T) {
		_, err := NewSafeAppName(istructs.AppQName_test1_app1, func(name string) (bool, error) { return false, nil })
		require.ErrorIs(err, ErrNoSafeAppName)
	})

	t.Run("unique func error", func(t *testing.T) {
		testErr := errors.New("test error")
		_, err := NewSafeAppName(istructs.AppQName_test1_app1, func(name string) (bool, error) { return true, testErr })
		require.ErrorIs(err, testErr)
	})
}

func TestSafeAppNameUnmarshalJSONErrors(t *testing.T) {
	require := require.New(t)
	san := SafeAppName{name: ""}
	err := san.UnmarshalJSON(nil)
	require.Error(err)
	log.Println(err)
}
