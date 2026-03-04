/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
)

// pipeline.IService
func (s *acmeService) Prepare(work interface{}) error {
	return nil
}

// pipeline.IService
func (s *acmeService) Run(ctx context.Context) {
	s.BaseContext = func(l net.Listener) context.Context {
		return ctx // need to track both client disconnect and app finalize
	}
	log.Println("Starting ACME HTTP server on :80")
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Println("ACME HTTP server failure: ", err.Error())
	}
}

// pipeline.IService
func (s *acmeService) Stop() {
	// ctx here is used to avoid eternal waiting for close idle connections and listeners
	// all connections and listeners are closed in the explicit way so it is not necessary to track ctx
	if err := s.Shutdown(context.Background()); err != nil {
		_ = s.Close()
	}
}
