/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/wneessen/go-mail"
)

type sendMailStorage struct {
	messagesSenderOverride chan smtptest.Message // not nil in tests only
}

func NewSendMailStorage(messages chan smtptest.Message) state.IStateStorage {
	return &sendMailStorage{
		messagesSenderOverride: messages,
	}
}

type mailKeyBuilder struct {
	baseKeyBuilder
	to       []string
	cc       []string
	bcc      []string
	host     string
	port     int32
	username string
	password string
	from     string
	subject  string
	body     string
}

func (b *mailKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*mailKeyBuilder)
	if !ok {
		return false
	}
	vb := src.(*mailKeyBuilder)
	if len(b.to) != len(vb.to) {
		return false
	}
	for i, v := range b.to {
		if v != vb.to[i] {
			return false
		}
	}
	if len(b.cc) != len(vb.cc) {
		return false
	}
	for i, v := range b.cc {
		if v != vb.cc[i] {
			return false
		}
	}
	if len(b.bcc) != len(vb.bcc) {
		return false
	}
	for i, v := range b.bcc {
		if v != vb.bcc[i] {
			return false
		}
	}
	if b.host != vb.host {
		return false
	}
	if b.port != vb.port {
		return false
	}
	if b.username != vb.username {
		return false
	}
	if b.password != vb.password {
		return false
	}
	if b.from != vb.from {
		return false
	}
	if b.subject != vb.subject {
		return false
	}
	if b.body != vb.body {
		return false
	}
	return true
}

func (b *mailKeyBuilder) PutString(name string, value string) {
	switch name {
	case sys.Storage_SendMail_Field_To:
		b.to = append(b.to, value)
	case sys.Storage_SendMail_Field_CC:
		b.cc = append(b.cc, value)
	case sys.Storage_SendMail_Field_BCC:
		b.bcc = append(b.bcc, value)
	case sys.Storage_SendMail_Field_Host:
		b.host = value
	case sys.Storage_SendMail_Field_Username:
		b.username = value
	case sys.Storage_SendMail_Field_Password:
		b.password = value
	case sys.Storage_SendMail_Field_From:
		b.from = value
	case sys.Storage_SendMail_Field_Subject:
		b.subject = value
	case sys.Storage_SendMail_Field_Body:
		b.body = value
	default:
		b.baseKeyBuilder.PutString(name, value)
	}
}

func (b *mailKeyBuilder) PutInt32(name string, value int32) {
	if name == sys.Storage_SendMail_Field_Port {
		b.port = value
	} else {
		b.baseKeyBuilder.PutInt32(name, value)
	}
}

type sendMailValueBuilder struct {
	baseValueBuilder
}

func (s *sendMailStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &mailKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_SendMail},
		to:             make([]string, 0),
		cc:             make([]string, 0),
		bcc:            make([]string, 0),
	}
}
func (s *sendMailStorage) Validate(items []state.ApplyBatchItem) (err error) {
	for _, item := range items {
		k := item.Key.(*mailKeyBuilder)

		notExists := func(field string) (err error) {
			return fmt.Errorf("'%s': %w", field, ErrNotFound)
		}
		if k.host == "" {
			return notExists(sys.Storage_SendMail_Field_Host)
		}
		if k.port == 0 {
			return notExists(sys.Storage_SendMail_Field_Port)
		}
		if k.from == "" {
			return notExists(sys.Storage_SendMail_Field_From)
		}
		if len(item.Key.(*mailKeyBuilder).to) == 0 {
			return fmt.Errorf("'%s': %w", sys.Storage_SendMail_Field_To, ErrNotFound)
		}
	}
	return nil
}
func (s *sendMailStorage) ApplyBatch(items []state.ApplyBatchItem) (err error) {
	stringOrEmpty := func(value string) string {
		if value != "" {
			return value
		}
		return ""
	}
	for _, item := range items {
		k := item.Key.(*mailKeyBuilder)

		msg := mail.NewMsg()
		msg.Subject(stringOrEmpty(k.subject))
		err = msg.From(k.from)
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
		msg.SetBodyString(mail.TypeTextHTML, stringOrEmpty(k.body))
		msg.SetCharset(mail.CharsetUTF8)

		opts := []mail.Option{
			mail.WithPort(int(k.port)),
			mail.WithUsername(k.username),
			mail.WithPassword(k.password),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
		}

		if coreutils.IsTest() {
			opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
		}

		logger.Info(fmt.Sprintf("send mail '%s' from '%s' to %s, cc %s, bcc %s", stringOrEmpty(k.subject), k.from, k.to, k.cc, k.bcc))

		if s.messagesSenderOverride != nil {
			// happens in tests only
			m := smtptest.Message{
				Subject: stringOrEmpty(k.subject),
				From:    k.from,
				To:      k.to,
				CC:      k.cc,
				BCC:     k.bcc,
				Body:    stringOrEmpty(k.body),
			}
			s.messagesSenderOverride <- m
		} else {
			c, e := mail.NewClient(k.host, opts...)
			if e != nil {
				return e
			}
			err = c.DialAndSend(msg)
			if err != nil {
				return err
			}
		}

		logger.Info(fmt.Sprintf("mail '%s' from '%s' to %s, cc %s, bcc %s successfully sent", stringOrEmpty(k.subject), k.from, k.to, k.cc, k.bcc))
	}
	return nil
}
func (s *sendMailStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &sendMailValueBuilder{}, nil
}
