/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys"
)

func TestSendMailStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	ts := smtptest.NewServer(smtptest.WithCredentials("user", "pwd"))
	defer ts.Close()
	storage := NewSendMailStorage(nil)
	k := storage.NewKeyBuilder(appdef.NullQName, nil)

	k.PutInt32(sys.Storage_SendMail_Field_Port, ts.Port())
	k.PutString(sys.Storage_SendMail_Field_Host, "localhost")
	k.PutString(sys.Storage_SendMail_Field_Username, "user")
	k.PutString(sys.Storage_SendMail_Field_Password, "pwd")

	k.PutString(sys.Storage_SendMail_Field_Subject, "Greeting")
	k.PutString(sys.Storage_SendMail_Field_From, "from@email.com")
	k.PutString(sys.Storage_SendMail_Field_To, "to0@email.com")
	k.PutString(sys.Storage_SendMail_Field_To, "to1@email.com")
	k.PutString(sys.Storage_SendMail_Field_CC, "cc0@email.com")
	k.PutString(sys.Storage_SendMail_Field_CC, "cc1@email.com")
	k.PutString(sys.Storage_SendMail_Field_BCC, "bcc0@email.com")
	k.PutString(sys.Storage_SendMail_Field_BCC, "bcc1@email.com")
	k.PutString(sys.Storage_SendMail_Field_Body, "Hello world")

	verifyMsg := func(msg smtptest.Message) {
		require.Equal("Greeting", msg.Subject)
		require.Equal("from@email.com", msg.From)
		require.Equal([]string{"to0@email.com", "to1@email.com"}, msg.To)
		require.Equal([]string{"cc0@email.com", "cc1@email.com"}, msg.CC)
		require.Equal([]string{"bcc0@email.com", "bcc1@email.com"}, msg.BCC)
		require.Equal("Hello world", msg.Body)
	}

	t.Run("Sending with Intent", func(t *testing.T) {
		v, err := storage.(state.IWithInsert).ProvideValueBuilder(k, nil)
		require.NoError(err)
		require.NotNil(v)
		err = storage.(state.IWithInsert).ApplyBatch([]state.ApplyBatchItem{{Key: k, Value: v}})
		require.NoError(err)
		msg := <-ts.Messages("user", "pwd")
		require.NotNil(msg)
		verifyMsg(msg)
	})

	t.Run("Sending with Get", func(t *testing.T) {
		v, err := storage.(state.IWithGet).Get(k)
		require.NoError(err)
		require.NotNil(v)
		require.True(v.AsBool(sys.Storage_SendMail_Field_Success))
		require.Empty(v.AsString(sys.Storage_SendMail_Field_ErrorMessage))
		msg := <-ts.Messages("user", "pwd")
		require.NotNil(msg)
		verifyMsg(msg)
	})
}

func TestSendMailStorage_Validate(t *testing.T) {
	tests := []struct {
		mandatoryField string
		kbFiller       func(kb istructs.IStateKeyBuilder)
	}{
		{
			mandatoryField: sys.Storage_SendMail_Field_Host,
			kbFiller:       func(kb istructs.IStateKeyBuilder) {},
		},
		{
			mandatoryField: sys.Storage_SendMail_Field_Port,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(sys.Storage_SendMail_Field_Host, "smtp.gmail.com")
			},
		},
		{
			mandatoryField: sys.Storage_SendMail_Field_From,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(sys.Storage_SendMail_Field_Host, "smtp.gmail.com")
				kb.PutInt32(sys.Storage_SendMail_Field_Port, 587)
				kb.PutString(sys.Storage_SendMail_Field_Username, "user")
				kb.PutString(sys.Storage_SendMail_Field_Password, "pwd")
			},
		},
		{
			mandatoryField: sys.Storage_SendMail_Field_To,
			kbFiller: func(kb istructs.IStateKeyBuilder) {
				kb.PutString(sys.Storage_SendMail_Field_Host, "smtp.gmail.com")
				kb.PutInt32(sys.Storage_SendMail_Field_Port, 587)
				kb.PutString(sys.Storage_SendMail_Field_Username, "user")
				kb.PutString(sys.Storage_SendMail_Field_Password, "pwd")
				kb.PutString(sys.Storage_SendMail_Field_From, "sender@email.com")
			},
		},
	}
	storage := NewSendMailStorage(nil)
	for _, test := range tests {
		t.Run(fmt.Sprintf("Send with intents: error when mandatory field '%s' not found", test.mandatoryField), func(t *testing.T) {
			require := require.New(t)
			k := storage.NewKeyBuilder(appdef.NullQName, nil)
			test.kbFiller(k)
			_, err := storage.(state.IWithInsert).ProvideValueBuilder(k, nil)
			require.NoError(err)
			err = storage.(state.IWithInsert).Validate([]state.ApplyBatchItem{{Key: k, Value: nil}})
			require.ErrorIs(err, ErrNotFound)
			require.Contains(err.Error(), test.mandatoryField)
		})
		t.Run(fmt.Sprintf("Send with Get: error when mandatory field '%s' not found", test.mandatoryField), func(t *testing.T) {
			require := require.New(t)
			k := storage.NewKeyBuilder(appdef.NullQName, nil)
			test.kbFiller(k)
			v, err := storage.(state.IWithGet).Get(k)
			require.NoError(err)
			require.NotNil(v)
			require.False(v.AsBool(sys.Storage_SendMail_Field_Success))
			require.Contains(v.AsString(sys.Storage_SendMail_Field_ErrorMessage), test.mandatoryField)
		})
	}
}
