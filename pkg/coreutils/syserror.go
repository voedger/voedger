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
	headers    map[string]string
}

func (he SysError) AddHeader(key, value string) SysError {
	if _, ok := he.headers[key]; ok {
		panic(fmt.Sprintf("header %q is already set", key))
	}
	if he.headers == nil {
		he.headers = map[string]string{}
	}
	he.headers[key] = value
	return he
}

func (he SysError) Headers() map[string]string {
	return he.headers
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

func (he SysError) Is(target error) bool {
	var t SysError
	switch v := target.(type) {
	case SysError:
		t = v
	case *SysError:
		if v == nil {
			return false
		}
		t = *v
	default:
		return false
	}
	return he.HTTPStatus == t.HTTPStatus && he.QName == t.QName && he.Message == t.Message && he.Data == t.Data
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

func (he SysError) IsNil() bool {
	return he.HTTPStatus == 0 && len(he.Data) == 0 && len(he.Message) == 0 && he.QName == appdef.NullQName
}

func NewHTTPErrorf(httpStatus int, args ...interface{}) SysError {
	return SysError{
		HTTPStatus: httpStatus,
		Message:    fmt.Sprint(args...),
	}
}

func NewHTTPError(httpStatus int, err error) SysError {
	return NewHTTPErrorf(httpStatus, err.Error())
}
