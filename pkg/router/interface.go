package router

import "github.com/voedger/voedger/pkg/pipeline"

type IHTTPService interface {
	pipeline.IService
	GetPort() int
}

type IACMEService interface {
	pipeline.IService
}
