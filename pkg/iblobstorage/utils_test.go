/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

import (
	"encoding/base64"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSUUID(t *testing.T) {
	suuid := NewSUUID()

	log.Println(suuid)

	const expectedLength = 43
	require.Len(t, suuid, expectedLength)

	require.Regexp(t, "[a-zA-Z0-9-_]", suuid)

	_, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(string(suuid))
	require.NoError(t, err)
}

func TestDurationSeconds(t *testing.T) {
	dt := DurationType(1)
	require.Equal(t, 86400, dt.Seconds())
	dt = DurationType(2)
	require.Equal(t, 86400*2, dt.Seconds())
}
