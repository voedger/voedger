/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
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

func ProvideService(serviceChannel iprocbus.ServiceChannel, blobStorage iblobstorage.IBLOBStorage,
	ibus ibus.IBus, busTimeout time.Duration) pipeline.IService {
	return pipeline.NewService(func(ctx context.Context) {
		for ctx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobMessage := workIntf.(IBLOBMessage)
				switch blobMessage.BLOBOperation() {
				case BLOBOperation_Read_Persistent, BLOBOperation_Read_Temporary:
					readBLOB(blobMessage, blobStorage, ibus, busTimeout)
				case BLOBOperation_Write_Persistent_Single:
					writeBLOB()
				}
			case <-ctx.Done():
			}
		}
	})
}
