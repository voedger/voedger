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
		{appdef.ErrMissed("msg %s", "par"), appdef.ErrMissedError, "msg par"},
		{appdef.ErrInvalid("msg %s", "par"), appdef.ErrInvalidError, "msg par"},
		{appdef.ErrOutOfBounds("msg %s", "par"), appdef.ErrOutOfBoundsError, "msg par"},
		{appdef.ErrAlreadyExists("msg %s", "par"), appdef.ErrAlreadyExistsError, "msg par"},
		{appdef.ErrNotFound("msg %s", "par"), appdef.ErrNotFoundError, "msg par"},
		{appdef.ErrFieldNotFound("field"), appdef.ErrNotFoundError, "field"},
		{appdef.ErrTypeNotFound(appdef.QNameANY), appdef.ErrNotFoundError, appdef.QNameANY.String()},
		{appdef.ErrRoleNotFound(appdef.QNameANY), appdef.ErrNotFoundError, appdef.QNameANY.String()},
		{appdef.ErrFilterHasNoMatches("test", nil, "ws"), appdef.ErrNotFoundError, "test"},
		{appdef.ErrConvert("msg %s", "par"), appdef.ErrConvertError, "msg par"},
		{appdef.ErrTooMany("msg %s", "par"), appdef.ErrTooManyError, "msg par"},
		{appdef.ErrIncompatible("msg %s", "par"), appdef.ErrIncompatibleError, "msg par"},
		{appdef.ErrUnsupported("msg %s", "par"), appdef.ErrUnsupportedError, "msg par"},
		{appdef.ErrACLUnsupportedType(appdef.NullType), appdef.ErrUnsupportedError, "null"},
	}

	require := require.New(t)
	for _, tt := range tests {
		require.Error(tt.e, require.Is(tt.is), require.Has(tt.has))
	}
}
