/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
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
	principalsFunc := func() []iauthnz.Principal { return principals }
	storage := NewSubjectStorage(principalsFunc, tokenFunc)
	k := storage.NewKeyBuilder(appdef.NullQName, nil)

	v, err := storage.(state.IWithGet).Get(k)
	require.NoError(err)

	require.Equal(int64(principals[0].WSID), v.AsInt64(sys.Storage_RequestSubject_Field_ProfileWSID))
	require.Equal(int32(istructs.SubjectKind_User), v.AsInt32(sys.Storage_RequestSubject_Field_Kind))
	require.Equal(principals[0].Name, v.AsString(sys.Storage_RequestSubject_Field_Name))
	require.Equal(token, v.AsString(sys.Storage_RequestSubject_Field_Token))
}
