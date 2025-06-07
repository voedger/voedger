/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package abstracts_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/abstracts"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_WithAbstract(t *testing.T) {
	require := require.New(t)

	a := abstracts.MakeWithAbstract()
	require.False(a.Abstract())

	abstracts.SetAbstract(&a)
	require.True(a.Abstract())

	// check interface compatibility
	var _ appdef.IWithAbstract = &a
}

func Test_WithAbstractBuilder(t *testing.T) {
	require := require.New(t)

	a := abstracts.MakeWithAbstract()
	require.False(a.Abstract())

	b := abstracts.MakeWithAbstractBuilder(&a)
	b.SetAbstract()
	require.True(a.Abstract())

	// check interface compatibility
	var _ appdef.IWithAbstractBuilder = &b
}
