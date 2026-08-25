/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

func (s *acmeService) Prepare(interface{}) error {
	return s.prepareBasicServer(s.handler)
}
