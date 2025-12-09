/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vit

import (
	"net/http"

	"github.com/voedger/voedger/pkg/goutils/httpu"
)

type vitOptsKey struct{}

type vitReqOpts struct {
	expectedMessages []string
}

func WithExpectedCode(code int, expectedMessages ...string) httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.WithExpectedCode(code))
		vitOpts := opts.CustomOpts(vitOptsKey{}).(*vitReqOpts)
		vitOpts.expectedMessages = append(vitOpts.expectedMessages, expectedMessages...)
	}
}

func createVITOpts() httpu.ReqOptFunc {
	return httpu.WithCustomOpts(vitOptsKey{}, &vitReqOpts{})
}

func Expect400RefIntegrity_Existence() httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.Expect400())
		vitOpts := opts.CustomOpts(vitOptsKey{}).(*vitReqOpts)
		vitOpts.expectedMessages = append(vitOpts.expectedMessages, "referential integrity violation", "does not exist")
	}
}

func Expect400RefIntegrity_QName() httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.Expect400())
		vitOpts := opts.CustomOpts(vitOptsKey{}).(*vitReqOpts)
		vitOpts.expectedMessages = append(vitOpts.expectedMessages, "referential integrity violation", "QNames are only allowed")
	}
}

func Expect400(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, expectedMessages...)
}

func Expect401(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusUnauthorized, expectedMessages...)
}

func Expect403(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusForbidden, expectedMessages...)
}

func Expect404(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusNotFound, expectedMessages...)
}

func Expect405(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusMethodNotAllowed, expectedMessages...)
}

func Expect409(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusConflict, expectedMessages...)
}

func Expect500(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusInternalServerError, expectedMessages...)
}
