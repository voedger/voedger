/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NullType(t *testing.T) {
	require := require.New(t)

	require.Empty(NullType.Comment())
	require.Empty(NullType.CommentLines())

	require.Nil(NullType.App())
	require.Equal(NullQName, NullType.QName())
	require.Equal(TypeKind_null, NullType.Kind())
	require.False(NullType.IsSystem())

	require.Contains(fmt.Sprint(NullType), "null type")
}

func Test_AnyType(t *testing.T) {
	require := require.New(t)

	require.Empty(AnyType.Comment())
	require.Empty(AnyType.CommentLines())

	require.Nil(AnyType.App())
	require.Equal(QNameANY, AnyType.QName())
	require.Equal(TypeKind_Any, AnyType.Kind())
	require.False(AnyType.IsSystem())

	require.Contains(fmt.Sprint(AnyType), "any type")
}
