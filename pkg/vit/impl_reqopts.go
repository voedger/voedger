/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vit

import (
	"net/http"

	"github.com/voedger/voedger/pkg/goutils/httpu"
)

type vitReqOpts struct {
	httpu.IReqOpts   // IReqOpts must be included, not defined as a field
	expectedMessages []string
}

// must be the first opt
func WithVITOpts() httpu.ReqOptFunc {
	return httpu.WithCustomOptsProvider(func(internalOpts httpu.IReqOpts) (customOpts httpu.IReqOpts) {
		return &vitReqOpts{IReqOpts: internalOpts}
	})
}

func WithExpectedCode(code int, expectedMessages ...string) httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.WithExpectedCode(code))
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, expectedMessages...)
	}
}

func Expect400RefIntegrity_Existence() httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.Expect400())
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, "referential integrity violation", "does not exist")
	}
}

func Expect400RefIntegrity_QName() httpu.ReqOptFunc {
	return func(opts httpu.IReqOpts) {
		opts.Append(httpu.Expect400())
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, "referential integrity violation", "QNames are only allowed")
	}
}

func Expect400(expectedMessages ...string) httpu.ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, expectedMessages...)
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
