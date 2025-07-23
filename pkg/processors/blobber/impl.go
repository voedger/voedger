/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/pipeline"
)

// [~server.apiv2.blobs/cmp.blobber.ServicePipeline~impl]
func providePipeline(vvmCtx context.Context, blobStorage iblobstorage.IBLOBStorage,
	wLimiterFactory WLimiterFactory) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "blob processor",
		pipeline.WireSyncOperator("switch", pipeline.SwitchOperator(&blobReadOrWriteSwitch{},
			pipeline.SwitchBranch(branchReadBLOB, pipeline.NewSyncPipeline(vvmCtx, branchReadBLOB,
				pipeline.WireFunc("getBLOBMessageRead", getBLOBMessageRead),                     // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_getBLOBMessageRead~impl]
				pipeline.WireFunc("getBLOBIDFromOwner", getBLOBIDFromOwner),                     // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_getBLOBIDFromOwner~impl]
				pipeline.WireFunc("getBLOBKeyRead", getBLOBKeyRead),                             // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_getBLOBKeyRead~impl]
				pipeline.WireFunc("queryBLOBState", provideQueryAndCheckBLOBState(blobStorage)), // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_queryBLOBState~impl]
				pipeline.WireFunc("downloadBLOBHelper", downloadBLOBHelper),                     // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_downloadBLOBHelper~impl]
				pipeline.WireFunc("initResponse", initResponse),                                 // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_initResponse~impl]
				pipeline.WireFunc("readBLOB", provideReadBLOB(blobStorage)),                     // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_readBLOB~impl]
				pipeline.WireSyncOperator("catchReadError", &catchReadError{}),                  // [~server.apiv2.blobs/cmp.blobber.ServicePipeline_catchReadError~impl]
			)),
			pipeline.SwitchBranch(branchWriteBLOB, pipeline.NewSyncPipeline(vvmCtx, branchWriteBLOB,
				pipeline.WireFunc("getBLOBMessageWrite", getBLOBMessageWrite),
				pipeline.WireFunc("parseQueryParams", parseQueryParams),
				pipeline.WireFunc("parseMediaType", parseMediaType),
				pipeline.WireFunc("validateQueryParams", validateQueryParams),
				pipeline.WireFunc("getRegisterFunc", getRegisterFunc),
				pipeline.WireSyncOperator("wrapBadRequest", &badRequestWrapper{}),
				pipeline.WireFunc("registerBLOB", registerBLOB),
				pipeline.WireFunc("getBLOBKeyWrite", getBLOBKeyWrite),
				pipeline.WireFunc("writeBLOB", provideWriteBLOB(blobStorage, wLimiterFactory)),
				pipeline.WireFunc("setBLOBStatusCompleted", setBLOBStatusCompleted),
				pipeline.WireSyncOperator("sendResult", &sendWriteResult{}),
			)),
		)),
	)
}

func (b *blobReadOrWriteSwitch) Switch(work interface{}) (branchName string, err error) {
	blobWorkpiece := work.(*blobWorkpiece)
	if _, ok := blobWorkpiece.blobMessage.(*implIBLOBMessage_Read); ok {
		return branchReadBLOB, nil
	}
	return branchWriteBLOB, nil
}

func (b *blobWorkpiece) isPersistent() bool {
	if _, ok := b.blobMessage.(*implIBLOBMessage_Write); ok {
		return len(b.ttl) == 0
	}
	return len(b.blobMessage.(*implIBLOBMessage_Read).existingBLOBIDOrSUUID) <= temporaryBLOBIDLenTreshold
}
