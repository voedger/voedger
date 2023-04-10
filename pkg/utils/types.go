/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import "github.com/untillpro/voedger/pkg/istructs"

type ErrorWrapperType func(err error, defaultStatusCode int) (httpError error)

type SysError struct {
	HTTPStatus int
	QName      istructs.QName
	Message    string
	Data       string
}
