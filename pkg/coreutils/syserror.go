/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
)

type SysError struct {
	HTTPStatus int
	QName      appdef.QName
	Message    string
	Data       string
}

func NewSysError(statusCode int) error {
	return SysError{HTTPStatus: statusCode}
}

func WrapSysErrorToExact(err error, defaultStatusCode int) SysError {
	if err == nil {
		return SysError{}
	}
	var res SysError
	if !errors.As(err, &res) {
		return SysError{Message: err.Error(), HTTPStatus: defaultStatusCode}
	}
	return res
}

func WrapSysError(err error, defaultStatusCode int) error {
	if err == nil {
		return err
	}
	return WrapSysErrorToExact(err, defaultStatusCode)
}

func (he SysError) Error() string {
	if len(he.Message) == 0 && he.HTTPStatus > 0 {
		return fmt.Sprintf("%d %s", he.HTTPStatus, http.StatusText(he.HTTPStatus))
	}
	return he.Message
}

func (he SysError) ToJSON_APIV1() string {
	b := bytes.NewBuffer(nil)
	fmt.Fprintf(b, `{"sys.Error":{"HTTPStatus":%d,"Message":%q`, he.HTTPStatus, he.Message)
	if he.QName != appdef.NullQName {
		fmt.Fprintf(b, `,"QName":"%s"`, he.QName.String())
	}
	if len(he.Data) > 0 {
		fmt.Fprintf(b, `,"Data":%q`, he.Data)
	}
	b.WriteString("}}")
	return b.String()
}

func (he SysError) ToJSON_APIV2() string {
	b := bytes.NewBufferString(fmt.Sprintf(`{"status":%d,"message":%q`, he.HTTPStatus, he.Message))
	if he.QName != appdef.NullQName {
		fmt.Fprintf(b, `,"qname":"%s"`, he.QName.String())
	}
	if len(he.Data) > 0 {
		fmt.Fprintf(b, `,"data":%q`, he.Data)
	}
	b.WriteString("}")
	return b.String()
}
