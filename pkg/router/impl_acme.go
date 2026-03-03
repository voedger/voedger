/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import "log"

// pipeline.IService
func (s *acmeService) Prepare(work interface{}) (err error) {
	if err := s.prepareBasicServer(); err != nil {
		return err
	}
	filteringLogger := log.New(&filteringWriter{log.Default().Writer()}, log.Default().Prefix(), log.Default().Flags())
	s.server.ErrorLog = filteringLogger
	return nil
}
