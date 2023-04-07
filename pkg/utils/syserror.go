/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/untillpro/voedger/pkg/istructs"
)

func NewSysError(statusCode int) error {
	return SysError{HTTPStatus: statusCode}
}

func WrapSysError(err error, defaultStatusCode int) error {
	if err == nil {
		return nil
	}
	var res SysError
	if !errors.As(err, &res) {
		res = SysError{Message: err.Error(), HTTPStatus: defaultStatusCode}
	}
	res.Message = strings.Replace(res.Message, "[exec function/doSync] ", "", -1) // TODO: how to remove this error part?
	res.Message = strings.Replace(res.Message, "[execCommand/doSync] ", "", -1)
	return res
}

func (he SysError) Error() string {
	return he.Message
}

func (he SysError) ToJSON() string {
	b := bytes.NewBuffer(nil)
	b.WriteString(fmt.Sprintf(`{"sys.Error":{"HTTPStatus":%d,"Message":%q`, he.HTTPStatus, he.Message))
	if he.QName != istructs.NullQName {
		b.WriteString(fmt.Sprintf(`,"QName":"%s"`, he.QName.String()))
	}
	if len(he.Data) > 0 {
		b.WriteString(fmt.Sprintf(`,"Data":%q`, he.Data))
	}
	b.WriteString("}}")
	return b.String()
}
