/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// pipeline.IService
func (s *acmeService) Prepare(work interface{}) error {
	return nil
}

// pipeline.IService
func (s *acmeService) Run(ctx context.Context) {
	s.preRun(ctx)
	s.log("starting on %s", s.listener.Addr().(*net.TCPAddr).String())
	if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		s.log("ListenAndServe() error: %s", err.Error())
	}
}

// // pipeline.IService
// func (s *acmeService) Stop() {
// 	// ctx here is used to avoid eternal waiting for close idle connections and listeners
// 	// all connections and listeners are closed in the explicit way so it is not necessary to track ctx
// 	if err := s.server.Shutdown(context.Background()); err != nil {
// 		_ = s.Close()
// 	}
// }
