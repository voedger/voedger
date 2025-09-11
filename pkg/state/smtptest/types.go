/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package smtptest

import (
	"errors"
	"io"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/voedger/voedger/pkg/state"
)

type Server interface {
	Port() int32
	Messages(username, password string) chan state.EmailMessage
	Close() error
}

type server struct {
	port     int32
	messages map[credentials]chan state.EmailMessage
	server   *smtp.Server
}

func (s *session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		if identity != "" && identity != username {
			return errors.New("invalid identity")
		}
		ch, ok := s.server.messages[credentials{
			username: username,
			password: password,
		}]
		if !ok {
			return errUnauthorized
		}
		s.ch = ch
		return nil
	}), nil
}

func (s *server) Port() int32 { return s.port }
func (s *server) Messages(username, password string) chan state.EmailMessage {
	return s.messages[credentials{
		username: username,
		password: password,
	}]
}
func (s *server) Close() error {
	for c := range s.messages {
		close(s.messages[c])
	}
	return s.server.Close()
}

type credentials struct {
	username string
	password string
}

type session struct {
	ch         chan state.EmailMessage
	recipients []string
	data       string
	server     *server
}

func (s *session) Reset() {}
func (s *session) Logout() error {
	s.ch <- s.message()
	return nil
}
func (s *session) Mail(_ string, _ *smtp.MailOptions) error { return nil }
func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.recipients = append(s.recipients, to)
	return nil
}
func (s *session) Data(r io.Reader) error {
	bb, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.data = string(bb)
	return nil
}
func (s *session) message() state.EmailMessage {
	ccMap := make(map[string]bool)
	toMap := make(map[string]bool)
	msg := state.EmailMessage{}
	var bodyStartLine int

	lines := strings.Split(s.data, "\r\n")
	for i, line := range lines {
		if line == "" {
			bodyStartLine = i + 1
			break
		}
		pair := strings.SplitN(line, ":", 2)
		switch pair[0] {
		case "Subject":
			msg.Subject = strings.TrimSpace(pair[1])
		case "From":
			msg.From = strings.Trim(pair[1], " <>")
		case "To":
			for _, to := range strings.Split(pair[1], ",") {
				to = strings.Trim(to, " <>")
				msg.To = append(msg.To, to)
				toMap[to] = true
			}
		case "Cc":
			for _, cc := range strings.Split(pair[1], ",") {
				cc = strings.Trim(cc, " <>")
				msg.CC = append(msg.CC, cc)
				ccMap[cc] = true
			}
		}
	}

	for _, recipient := range s.recipients {
		if toMap[recipient] {
			continue
		}
		if ccMap[recipient] {
			continue
		}
		msg.BCC = append(msg.BCC, recipient)
	}

	body := strings.Builder{}
	for i := bodyStartLine; i < len(lines); i++ {
		body.WriteString(lines[i])
	}
	msg.Body = body.String()

	return msg
}

type Option func(s Server)

func WithCredentials(username, password string) Option {
	return func(s Server) {
		s.(*server).messages[credentials{
			username: username,
			password: password,
		}] = make(chan state.EmailMessage, defaultMessagesChannelSize)
	}
}

type backend struct {
	server *server
}
