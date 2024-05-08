/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import "github.com/voedger/voedger/pkg/pipeline"

type IHTTPService interface {
	pipeline.IService
	GetPort() int
}

type IACMEService interface {
	pipeline.IService
}

type IAdminService IHTTPService