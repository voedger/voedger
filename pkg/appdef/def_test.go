/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NullDef(t *testing.T) {
	require := require.New(t)

	require.Nil(NullDef.App())
	require.Equal(NullQName, NullDef.QName())
	require.Equal(DefKind_null, NullDef.Kind())
}
