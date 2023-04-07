/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package isecrets

import "github.com/stretchr/testify/mock"

type SecretReaderMock struct {
	mock.Mock
}

func (m *SecretReaderMock) ReadSecret(name string) (bb []byte, err error) {
	aa := m.Called(name)
	if intf := aa.Get(0); intf != nil {
		bb = intf.([]byte)
	}
	err = aa.Error(1)
	return
}
