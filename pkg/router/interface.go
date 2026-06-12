/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import "github.com/voedger/voedger/pkg/pipeline"

type IHTTPService interface {
	pipeline.IServiceEx
	GetPort() int
}

type IACMEService interface {
	pipeline.IServiceEx
}

type IAdminService IHTTPService
