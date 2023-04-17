/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibusmem

import (
	"fmt"
	"time"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/ibus"
)

func New(params ibus.CLIParams) (impl ibus.IBus, cleanup func()) {
	bus := bus{maxNumOfConcurrentRequests: params.MaxNumOfConcurrentRequests, readWriteTimeout: params.ReadWriteTimeout, addressHandlersMap: make(map[addressType]*addressHandlerType)}

	bus.requestContextsPool = make(chan *requestContextType, params.MaxNumOfConcurrentRequests)
	for i := 0; i < params.MaxNumOfConcurrentRequests; i++ {
		requestContext := requestContextType{
			errReached:      true,
			responseChannel: make(responseChannelType, ResponseChannelBufferSize),
			senderTimer:     time.NewTimer(params.ReadWriteTimeout),
		}
		bus.requestContextsPool <- &requestContext
	}
	logger.Info("bus started:", fmt.Sprintf("%#v", params))
	return &bus, bus.cleanup
}
