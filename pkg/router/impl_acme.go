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
