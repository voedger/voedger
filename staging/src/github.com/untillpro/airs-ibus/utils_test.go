/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package ibus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateResponse(t *testing.T) {
	r := CreateResponse(1, "test")
	require.Equal(t, 1, r.StatusCode)
	require.Equal(t, "test", string(r.Data))
	require.Empty(t, r.ContentType)
}

func TestCreateErrorResponse(t *testing.T) {
	r := CreateErrorResponse(1, errors.New("test"))
	require.Equal(t, 1, r.StatusCode)
	require.Equal(t, "test", string(r.Data))
	require.Equal(t, "text/plain", r.ContentType)
}
