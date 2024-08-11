/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestSubjectStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	principals := []iauthnz.Principal{{
		Kind: iauthnz.PrincipalKind_User,
		WSID: 42,
		Name: "john.doe@acme.com",
	}}
	token := "token"
	tokenFunc := func() string { return token }
	s := ProvideCommandProcessorStateFactory()(context.Background(), func() istructs.IAppStructs { return &nilAppStructs{} },
		nil, nil, nil, nil, func() []iauthnz.Principal { return principals }, tokenFunc, 1, nil, nil, nil, nil, nil)
	k, err := s.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
	require.NoError(err)

	v, err := s.MustExist(k)
	require.NoError(err)

	require.Equal(int64(principals[0].WSID), v.AsInt64(sys.Storage_RequestSubject_Field_ProfileWSID))
	require.Equal(int32(istructs.SubjectKind_User), v.AsInt32(sys.Storage_RequestSubject_Field_Kind))
	require.Equal(principals[0].Name, v.AsString(sys.Storage_RequestSubject_Field_Name))
	require.Equal(token, v.AsString(sys.Storage_RequestSubject_Field_Token))
}
