/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func ProvideService(serviceChannel BLOBServiceChannel, blobStorage iblobstorage.IBLOBStorage,
	ibus ibus.IBus, busTimeout time.Duration, wLimiterFactory WLimiterFactory) pipeline.IService {
	return pipeline.NewService(func(vvmCtx context.Context) {
		pipeline := providePipeline(vvmCtx, blobStorage, ibus, busTimeout, wLimiterFactory)
		for vvmCtx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobWorkpiece := &blobWorkpiece{
					blobMessage: workIntf.(iBLOBMessage_Base),
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

func NewIRequestHandler(procbus iprocbus.IProcBus, chanGroupIdx BLOBServiceChannelGroupIdx) IRequestHandler {
	return &implIRequestHandler{
		procbus:      procbus,
		chanGroupIdx: chanGroupIdx,
	}
}
