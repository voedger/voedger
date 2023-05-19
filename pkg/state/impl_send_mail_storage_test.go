/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
)

func TestSendMailStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	ts := smtptest.NewServer(smtptest.WithCredentials("user", "pwd"))
	defer ts.Close()
	s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 1, 0)
	k, err := s.KeyBuilder(SendMailStorage, appdef.NullQName)
	require.NoError(err)

	k.PutInt32(Field_Port, ts.Port())
	k.PutString(Field_Host, "localhost")
	k.PutString(Field_Username, "user")
	k.PutString(Field_Password, "pwd")

	k.PutString(Field_Subject, "Greeting")
	k.PutString(Field_From, "from@email.com")
	k.PutString(Field_To, "to0@email.com")
	k.PutString(Field_To, "to1@email.com")
	k.PutString(Field_CC, "cc0@email.com")
	k.PutString(Field_CC, "cc1@email.com")
	k.PutString(Field_BCC, "bcc0@email.com")
	k.PutString(Field_BCC, "bcc1@email.com")
	k.PutString(Field_Body, "Hello world")

	require.Nil(s.NewValue(k))
	readyToFlush, err := s.ApplyIntents()
	require.True(readyToFlush)
	require.NoError(err)
	require.NoError(s.FlushBundles())

	msg := <-ts.Messages("user", "pwd")

	require.Equal("Greeting", msg.Subject)
	require.Equal("from@email.com", msg.From)
	require.Equal([]string{"to0@email.com", "to1@email.com"}, msg.To)
	require.Equal([]string{"cc0@email.com", "cc1@email.com"}, msg.CC)
	require.Equal([]string{"bcc0@email.com", "bcc1@email.com"}, msg.BCC)
	require.Equal("Hello world", msg.Body)
}

func TestSendMailStorage_Validate(t *testing.T) {
	tests := []struct {
		mandatoryField string
		kbFiller       func(kb istructs.IStateKeyBuilder)
	}{
		{
			mandatoryField: Field_Host,
			kbFiller:       func(kb istructs.IStateKeyBuilder) {},
		},
		{
			mandatoryField: Field_Port,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(Field_Host, "smtp.gmail.com")
			},
		},
		{
			mandatoryField: Field_Username,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(Field_Host, "smtp.gmail.com")
				kb.PutInt32(Field_Port, 587)
			},
		},
		{
			mandatoryField: Field_Password,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(Field_Host, "smtp.gmail.com")
				kb.PutInt32(Field_Port, 587)
				kb.PutString(Field_Username, "user")
			},
		},
		{
			mandatoryField: Field_From,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(Field_Host, "smtp.gmail.com")
				kb.PutInt32(Field_Port, 587)
				kb.PutString(Field_Username, "user")
				kb.PutString(Field_Password, "pwd")
			},
		},
		{
			mandatoryField: Field_To,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(Field_Host, "smtp.gmail.com")
				kb.PutInt32(Field_Port, 587)
				kb.PutString(Field_Username, "user")
				kb.PutString(Field_Password, "pwd")
				kb.PutString(Field_From, "sender@email.com")
			},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("Should return error when mandatory field '%s' not found", test.mandatoryField), func(t *testing.T) {
			require := require.New(t)
			s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 1, 0)
			k, err := s.KeyBuilder(SendMailStorage, appdef.NullQName)
			require.NoError(err)
			test.kbFiller(k)
			_, err = s.NewValue(k)
			require.NoError(err)

			readyToFlush, err := s.ApplyIntents()

			require.False(readyToFlush)
			require.ErrorIs(err, ErrNotFound)
			require.Contains(err.Error(), test.mandatoryField)
		})
	}
}
