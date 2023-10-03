/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestPrincipalTokenStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	principals := []iauthnz.Principal{{
		Kind: iauthnz.PrincipalKind_User,
		WSID: 42,
		Name: "john.doe@acme.com",
	}}
	token := "token"
	tokenFunc := func() string { return token }
	s := ProvideCommandProcessorStateFactory()(context.Background(), func() istructs.IAppStructs { return &nilAppStructs{} }, nil, nil, nil, nil, func() []iauthnz.Principal { return principals }, tokenFunc, 1, nil)
	k, err := s.KeyBuilder(PrincipalTokenStorage, appdef.NullQName)
	require.NoError(err)

	v, err := s.MustExist(k)
	require.NoError(err)

	require.Equal(token, v.AsString(Field_Token))
}
