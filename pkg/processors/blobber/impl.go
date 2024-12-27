/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/pipeline"
)

func ProvideService(serviceChannel iprocbus.ServiceChannel, blobStorage iblobstorage.IBLOBStorage) pipeline.IService {
	return pipeline.NewService(func(ctx context.Context) {
		for ctx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobMessage := workIntf.(IBLOBMessage)
				switch blobMessage.BLOBOperation() {
				case BLOBOperation_Read_Persistent:

				}
			case <-ctx.Done():
			}
		}
	})
}
