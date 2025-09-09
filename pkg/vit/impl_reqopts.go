/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vit

import (
	"net/http"

	"github.com/voedger/voedger/pkg/coreutils"
)

type vitReqOpts struct {
	coreutils.IReqOpts // IReqOpts must be included, not defined as a field
	expectedMessages []string
}

// must be the first opt
func WithVITOpts() coreutils.ReqOptFunc {
	return coreutils.WithCustomOptsProvider(func(internalOpts coreutils.IReqOpts) (customOpts coreutils.IReqOpts) {
		return &vitReqOpts{IReqOpts: internalOpts}
	})
}

func WithExpectedCode(code int, expectedMessages ...string) coreutils.ReqOptFunc {
	return func(opts coreutils.IReqOpts) {
		opts.Append(coreutils.WithExpectedCode(code))
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, expectedMessages...)
	}
}

func Expect400RefIntegrity_Existence() coreutils.ReqOptFunc {
	return func(opts coreutils.IReqOpts) {
		opts.Append(coreutils.Expect400())
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, "referential integrity violation", "does not exist")
	}
}

func Expect400RefIntegrity_QName() coreutils.ReqOptFunc {
	return func(opts coreutils.IReqOpts) {
		opts.Append(coreutils.Expect400())
		opts.(*vitReqOpts).expectedMessages = append(opts.(*vitReqOpts).expectedMessages, "referential integrity violation", "QNames are only allowed")
	}
}

func Expect400(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, expectedMessages...)
}

func Expect403(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusForbidden, expectedMessages...)
}

func Expect404(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusNotFound, expectedMessages...)
}

func Expect405(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusMethodNotAllowed, expectedMessages...)
}

func Expect409(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusConflict, expectedMessages...)
}

func Expect500(expectedMessages ...string) coreutils.ReqOptFunc {
	return WithExpectedCode(http.StatusInternalServerError, expectedMessages...)
}
