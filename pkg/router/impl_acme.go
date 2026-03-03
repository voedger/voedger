/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"net"

	"github.com/voedger/voedger/pkg/goutils/httpu"
)

// pipeline.IService
func (s *acmeService) Prepare(work interface{}) (err error) {
	if s.listener, err = net.Listen("tcp", httpu.ListenAddr(ACMEPort)); err != nil {
		return err
	}
	s.listeningPort.Store(ACMEPort)
	return nil
}
