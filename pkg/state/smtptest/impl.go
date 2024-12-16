/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package smtptest

import (
	"log"
	"net"

	"github.com/emersion/go-smtp"
	"github.com/voedger/voedger/pkg/coreutils"
)

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{ch: make(chan Message), server: b.server}, nil
}

func NewServer(opts ...Option) Server {
	ts := &server{messages: make(map[credentials]chan Message)}
	s := smtp.NewServer(&backend{server: ts})
	ts.server = s

	for _, opt := range opts {
		opt(ts)
	}

	l, err := net.Listen("tcp", coreutils.ServerAddress(0))
	if err != nil {
		panic(err)
	}
	ts.port = int32(l.Addr().(*net.TCPAddr).Port) // nolint G115

	s.AllowInsecureAuth = true

	go func() {
		log.Println("Starting test SMTP server at port", ts.port)
		if err = s.Serve(l); err != nil {
			log.Fatal(err)
		}
	}()

	return ts
}
