/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/pipeline"
)

func ProvideService(serviceChannel BLOBServiceChannel, blobStorage iblobstorage.IBLOBStorage, wLimiterFactory WLimiterFactory) pipeline.IService {
	return pipeline.NewService(func(vvmCtx context.Context) {
		pipeline := providePipeline(vvmCtx, blobStorage, wLimiterFactory)
		for vvmCtx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobWorkpiece := &blobWorkpiece{}
				switch typed := workIntf.(type) {
				case *implIBLOBMessage_Read:
					blobWorkpiece.blobMessage = typed
				case *implIBLOBMessage_Write:
					blobWorkpiece.blobMessage = typed
				}
				if err := pipeline.SendSync(blobWorkpiece); err != nil {
					// notest
					panic(err)
				}
				blobWorkpiece.Release()
			case <-vvmCtx.Done():
			}
		}
		pipeline.Close()
	})
}

func NewIRequestHandler(procbus iprocbus.IProcBus, chanGroupIdx BLOBServiceChannelGroupIdx, appParts appparts.IAppPartitions) IRequestHandler {
	return &implIRequestHandler{
		procbus:      procbus,
		chanGroupIdx: chanGroupIdx,
		appParts:     appParts,
	}
}
