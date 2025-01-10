/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_NullType(t *testing.T) {
	require := require.New(t)

	var _ appdef.IType = appdef.NullType // compile-time check

	require.Empty(appdef.NullType.Comment())
	require.Empty(slices.Collect(appdef.NullType.CommentLines()))

	require.False(appdef.NullType.HasTag(appdef.NullQName))
	appdef.NullType.Tags()(func(appdef.ITag) bool { require.Fail("Tags() should be empty"); return false })

	require.Nil(appdef.NullType.App())
	require.Nil(appdef.NullType.Workspace())
	require.Equal(appdef.NullQName, appdef.NullType.QName())
	require.Equal(appdef.TypeKind_null, appdef.NullType.Kind())
	require.False(appdef.NullType.IsSystem())

	require.Contains(fmt.Sprint(appdef.NullType), "null type")
}

func TestNullFields(t *testing.T) {
	require := require.New(t)

	var _ appdef.IWithFields = appdef.NullFields // compile-time check

	require.Nil(appdef.NullFields.Field("field"))
	require.Zero(appdef.NullFields.FieldCount())
	require.Empty(slices.Collect(appdef.NullFields.Fields()))

	require.Nil(appdef.NullFields.RefField("field"))
	require.Empty(appdef.NullFields.RefFields())

	require.Zero(appdef.NullFields.UserFieldCount())
	require.Empty(appdef.NullFields.UserFields())
}
