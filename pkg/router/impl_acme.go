/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import "log"

func (s *acmeService) Prepare(work interface{}) (err error) {
	filteringLogger := log.New(&filteringWriter{log.Default().Writer()}, log.Default().Prefix(), log.Default().Flags())
	if err = s.prepareBasicServer(s.handler); err == nil {
		s.server.ErrorLog = filteringLogger
	}
	return err
}
