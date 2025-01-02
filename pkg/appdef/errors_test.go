/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		e   error
		is  error
		has string
	}{
		{appdef.ErrMissed("field «%s»", "name"), appdef.ErrMissedError, "field «name»"},
		{appdef.ErrInvalid("field «%s»", "name"), appdef.ErrInvalidError, "field «name»"},
		{appdef.ErrOutOfBounds("%v percent", 101), appdef.ErrOutOfBoundsError, "101 percent"},
		{appdef.ErrAlreadyExists("this %v", "message"), appdef.ErrAlreadyExistsError, "this message"},
		{appdef.ErrNotFound("%v", 1), appdef.ErrNotFoundError, "1"},
	}

	require := require.New(t)
	for _, tt := range tests {
		require.Error(tt.e, require.Is(tt.is), require.Has(tt.has))
	}
}
