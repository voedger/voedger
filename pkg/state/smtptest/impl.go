/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package smtptest

import (
	"log"
	"net"

	"github.com/emersion/go-smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func NewServer(opts ...Option) Server {
	ts := &server{messages: make(map[credentials]chan Message)}
	s := smtp.NewServer(ts)
	ts.server = s

	for _, opt := range opts {
		opt(ts)
	}

	l, err := net.Listen("tcp", coreutils.ServerAddress(0))
	if err != nil {
		panic(err)
	}
	ts.port = l.Addr().(*net.TCPAddr).Port

	s.AllowInsecureAuth = true

	go func() {
		log.Println("Starting test SMTP server at port", ts.port)
		if err = s.Serve(l); err != nil {
			log.Fatal(err)
		}
	}()

	return ts
}
