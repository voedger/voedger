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

	expectedLength := 43
	require.Len(t, suuid, expectedLength)

	_, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(string(suuid))
	require.NoError(t, err)
}
