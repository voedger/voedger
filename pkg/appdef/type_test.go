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

	app := New()

	var any IType = &anyType{app: app}

	require.Empty(any.Comment())
	require.Empty(any.CommentLines())

	require.Equal(app, any.App())
	require.Equal(QNameANY, any.QName())
	require.Equal(TypeKind_Any, any.Kind())
	require.True(any.IsSystem())

	require.Contains(fmt.Sprint(any), "any type")
}
