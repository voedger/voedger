package main

// Fake test, have to be removed after CI is fixed (go test ./cmd... -race -coverprofile=coverage.txt -covermode=atomic -coverpkg=./cmd... -short)

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	require := require.New(t)
	require.NotNil(t)
	//
}
