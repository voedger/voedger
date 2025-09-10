/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/wneessen/go-mail"
)

type sendMailStorage struct {
	emailSender state.IEmailSender
	// messagesSenderOverride chan smtptest.Message // not nil in tests only
}

type implIEmailSender_SMTP struct {
	defaultOpts []mail.Option
}

func NewSendMailStorage(emailSender state.IEmailSender) state.IStateStorage {
	return &sendMailStorage{
		emailSender: emailSender,
	}
}

func NewIEmailSenderSMTP() state.IEmailSender {
	return &implIEmailSender_SMTP{}
}

func NewIEmailSenderSMTPForTests() state.IEmailSender {
	emailSender := NewIEmailSenderSMTP()
	emailSender.(*implIEmailSender_SMTP).defaultOpts = []mail.Option{mail.WithTLSPolicy(mail.NoTLS)}
	return emailSender
}

type mailKeyBuilder struct {
	baseKeyBuilder
	message  state.EmailMessage
	host     string
	port     int32
	username string
	password string
}

func (b *mailKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*mailKeyBuilder)
	if !ok {
		return false
	}
	vb := src.(*mailKeyBuilder)
	if len(b.message.To) != len(vb.message.To) {
		return false
	}
	for i, v := range b.message.To {
		if v != vb.message.To[i] {
			return false
		}
	}
	if len(b.message.CC) != len(vb.message.CC) {
		return false
	}
	for i, v := range b.message.CC {
		if v != vb.message.CC[i] {
			return false
		}
	}
	if len(b.message.BCC) != len(vb.message.BCC) {
		return false
	}
	for i, v := range b.message.BCC {
		if v != vb.message.BCC[i] {
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
	if b.message.From != vb.message.From {
		return false
	}
	if b.message.Subject != vb.message.Subject {
		return false
	}
	if b.message.Body != vb.message.Body {
		return false
	}
	return true
}

func (b *mailKeyBuilder) PutString(name string, value string) {
	switch name {
	case sys.Storage_SendMail_Field_To:
		b.message.To = append(b.message.To, value)
	case sys.Storage_SendMail_Field_CC:
		b.message.CC = append(b.message.CC, value)
	case sys.Storage_SendMail_Field_BCC:
		b.message.BCC = append(b.message.BCC, value)
	case sys.Storage_SendMail_Field_Host:
		b.host = value
	case sys.Storage_SendMail_Field_Username:
		b.username = value
	case sys.Storage_SendMail_Field_Password:
		b.password = value
	case sys.Storage_SendMail_Field_From:
		b.message.From = value
	case sys.Storage_SendMail_Field_Subject:
		b.message.Subject = value
	case sys.Storage_SendMail_Field_Body:
		b.message.Body = value
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
	}
}
func (s *sendMailStorage) validateKey(k *mailKeyBuilder) (err error) {
	const errMsg = "'%s': %w"
	if k.host == "" {
		return fmt.Errorf(errMsg, sys.Storage_SendMail_Field_Host, ErrNotFound)
	}
	if k.port == 0 {
		return fmt.Errorf(errMsg, sys.Storage_SendMail_Field_Port, ErrNotFound)
	}
	if k.message.From == "" {
		return fmt.Errorf(errMsg, sys.Storage_SendMail_Field_From, ErrNotFound)
	}
	if len(k.message.To) == 0 {
		return fmt.Errorf(errMsg, sys.Storage_SendMail_Field_To, ErrNotFound)
	}
	return nil
}
func (s *sendMailStorage) Validate(items []state.ApplyBatchItem) (err error) {
	for _, item := range items {
		if err := s.validateKey(item.Key.(*mailKeyBuilder)); err != nil {
			return err
		}
	}
	return nil
}
func (s *sendMailStorage) sendMail(k *mailKeyBuilder) error {

	opts := []mail.Option{
		mail.WithPort(int(k.port)),
		mail.WithUsername(k.username),
		mail.WithPassword(k.password),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
	}

	logger.Info(fmt.Sprintf("send mail '%s' from '%s' to %s, cc %s, bcc %s",
		k.message.Subject, k.message.From, k.message.To, k.message.CC, k.message.BCC))

	if err := s.emailSender.Send(k.host, k.message, opts...); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("mail '%s' from '%s' to %s, cc %s, bcc %s successfully sent",
		k.message.Subject, k.message.From, k.message.To, k.message.CC, k.message.BCC))
	return nil
}
func (s *sendMailStorage) ApplyBatch(items []state.ApplyBatchItem) (err error) {
	for _, item := range items {
		err = s.sendMail(item.Key.(*mailKeyBuilder))
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *sendMailStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &sendMailValueBuilder{}, nil
}

type sendMailValue struct {
	baseStateValue
	success bool
	error   string
}

func (v *sendMailValue) AsBool(name string) bool {
	switch name {
	case sys.Storage_SendMail_Field_Success:
		return v.success
	default:
		return v.baseStateValue.AsBool(name)
	}
}

func (v *sendMailValue) AsString(name string) string {
	switch name {
	case sys.Storage_SendMail_Field_ErrorMessage:
		return v.error
	default:
		return v.baseStateValue.AsString(name)
	}
}

func (s *sendMailStorage) Get(key istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	err := s.validateKey(key.(*mailKeyBuilder))
	if err == nil {
		err = s.sendMail(key.(*mailKeyBuilder))
	}
	if err != nil {
		return &sendMailValue{
			success: false,
			error:   err.Error(),
		}, nil
	}
	return &sendMailValue{success: true}, nil
}

func (s *implIEmailSender_SMTP) Send(host string, m state.EmailMessage, opts ...mail.Option) error {
	opts = append(opts, s.defaultOpts...)
	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return err
	}
	msg := mail.NewMsg()
	msg.Subject(m.Subject)
	if err = msg.From(m.From); err != nil {
		return err
	}
	if err = msg.To(m.To...); err != nil {
		return err
	}
	if err = msg.Cc(m.CC...); err != nil {
		return err
	}
	if err = msg.Bcc(m.BCC...); err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextHTML, m.Body)
	msg.SetCharset(mail.CharsetUTF8)
	return client.DialAndSend(msg)
}
