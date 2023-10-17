/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/wneessen/go-mail"
)

type sendMailStorage struct {
	messages chan smtptest.Message // not nil in tests only
}

func (s *sendMailStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &sendMailKeyBuilder{
		keyBuilder: newKeyBuilder(SendMail, appdef.NullQName),
		to:         make([]string, 0),
		cc:         make([]string, 0),
		bcc:        make([]string, 0),
	}
}
func (s *sendMailStorage) Validate(items []ApplyBatchItem) (err error) {
	for _, item := range items {
		k := item.key.(*sendMailKeyBuilder)

		mustExist := func(field string) (err error) {
			_, ok := k.data[field]
			if !ok {
				return fmt.Errorf("'%s': %w", field, ErrNotFound)
			}
			return
		}

		err = mustExist(Field_Host)
		if err != nil {
			return
		}
		err = mustExist(Field_Port)
		if err != nil {
			return
		}
		err = mustExist(Field_Username)
		if err != nil {
			return
		}
		err = mustExist(Field_Password)
		if err != nil {
			return
		}
		err = mustExist(Field_From)
		if err != nil {
			return
		}
		if len(item.key.(*sendMailKeyBuilder).to) == 0 {
			return fmt.Errorf("'%s': %w", Field_To, ErrNotFound)
		}
	}
	return nil
}
func (s *sendMailStorage) ApplyBatch(items []ApplyBatchItem) (err error) {
	stringOrEmpty := func(k *sendMailKeyBuilder, name string) string {
		if intf, ok := k.data[name]; ok {
			return intf.(string)
		}
		return ""
	}
	for _, item := range items {
		k := item.key.(*sendMailKeyBuilder)

		msg := mail.NewMsg()
		msg.Subject(stringOrEmpty(k, Field_Subject))
		err = msg.From(k.data[Field_From].(string))
		if err != nil {
			return
		}
		err = msg.To(k.to...)
		if err != nil {
			return
		}
		err = msg.Cc(k.cc...)
		if err != nil {
			return
		}
		err = msg.Bcc(k.bcc...)
		if err != nil {
			return
		}
		msg.SetBodyString(mail.TypeTextHTML, stringOrEmpty(k, Field_Body))
		msg.SetCharset(mail.CharsetUTF8)

		opts := []mail.Option{
			mail.WithPort(int(k.data[Field_Port].(int32))),
			mail.WithUsername(k.data[Field_Username].(string)),
			mail.WithPassword(k.data[Field_Password].(string)),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
		}

		if coreutils.IsTest() {
			opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
		}

		logger.Info(fmt.Sprintf("send mail '%s' from '%s' to %s, cc %s, bcc %s", stringOrEmpty(k, Field_Subject), k.data[Field_From], k.to, k.cc, k.bcc))

		if s.messages != nil {
			m := smtptest.Message{
				Subject: stringOrEmpty(k, Field_Subject),
				From:    k.data[Field_From].(string),
				To:      k.to,
				CC:      k.cc,
				BCC:     k.bcc,
				Body:    stringOrEmpty(k, Field_Body),
			}
			select {
			case s.messages <- m:
			default:
				// asumming VIT will be failed on TearDown
			}
		} else {
			c, e := mail.NewClient(k.data[Field_Host].(string), opts...)
			if e != nil {
				return e
			}
			err = c.DialAndSend(msg)
			if err != nil {
				return err
			}
		}

		logger.Info(fmt.Sprintf("mail '%s' from '%s' to %s, cc %s, bcc %s successfully sent", stringOrEmpty(k, Field_Subject), k.data[Field_From], k.to, k.cc, k.bcc))
	}
	return nil
}
func (s *sendMailStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return nil
}
