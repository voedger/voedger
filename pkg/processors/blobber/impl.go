/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func providePipeline(vvmCtx context.Context, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration,
	wLimiterFactory WLimiterFactory) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "blob processor",
		pipeline.WireSyncOperator("switch", pipeline.SwitchOperator(&blobOpSwitch{},
			pipeline.SwitchBranch(branchReadBLOB, pipeline.NewSyncPipeline(vvmCtx, branchReadBLOB,
				pipeline.WireFunc("getBLOBMessageRead", getBLOBMessageRead),
				pipeline.WireFunc("downloadBLOBHelper", provideDownloadBLOBHelper(bus, busTimeout)),
				pipeline.WireFunc("getBLOBKeyRead", getBLOBKeyRead),
				pipeline.WireFunc("queryBLOBState", provideQueryAndCheckBLOBState(blobStorage)),
				pipeline.WireFunc("initResponse", initResponse),
				pipeline.WireFunc("readBLOB", provideReadBLOB(blobStorage)),
				pipeline.WireSyncOperator("catchReadError", &catchReadError{}),
			)),
			pipeline.SwitchBranch(branchWriteBLOB, pipeline.NewSyncPipeline(vvmCtx, branchWriteBLOB,
				pipeline.WireFunc("getBLOBMessageWrite", getBLOBMessageWrite),
				pipeline.WireFunc("parseQueryParams", parseQueryParams),
				pipeline.WireFunc("parseMediaType", parseMediaType),
				pipeline.WireFunc("validateQueryParams", validateQueryParams),
				pipeline.WireFunc("getRegisterFuncName", getRegisterFuncName),
				pipeline.WireSyncOperator("wrapBadRequest", &badRequestWrapper{}),
				pipeline.WireFunc("registerBLOB", provideRegisterBLOB(bus, busTimeout)),
				pipeline.WireFunc("getBLOBKeyWrite", getBLOBKeyWrite),
				pipeline.WireFunc("writeBLOB", provideWriteBLOB(blobStorage, wLimiterFactory)),
				pipeline.WireFunc("setBLOBStatusCompleted", provideSetBLOBStatusCompleted(bus, busTimeout)),
				pipeline.WireSyncOperator("sendResult", &sendWriteResult{}),
			)),
		)),
	)
}

func (b *blobOpSwitch) Switch(work interface{}) (branchName string, err error) {
	blobWorkpiece := work.(*blobWorkpiece)
	if _, ok := blobWorkpiece.blobMessage.(IBLOBMessage_Read); ok {
		return branchReadBLOB, nil
	}
	return branchWriteBLOB, nil
}

func (b *blobWorkpiece) isPersistent() bool {
	if b.blobMessageWrite != nil {
		return len(b.ttl) == 0
	}
	return len(b.blobMessageRead.ExistingBLOBIDOrSUUID()) <= temporaryBLOBIDLenTreshold
}
