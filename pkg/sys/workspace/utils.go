/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

func validateWSKindInitializationData(as istructs.IAppStructs, data map[string]interface{}, t appdef.IType) (err error) {
	reb := as.Events().GetNewRawEventBuilder(
		istructs.NewRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				QName: t.QName(),
			},
		},
	)
	aob := reb.ArgumentObjectBuilder()
	aob.PutQName(appdef.SystemField_QName, t.QName())
	aob.PutRecordID(appdef.SystemField_ID, 1)
	if err = coreutils.MapToObject(data, aob); err != nil {
		return err
	}
	_, err = aob.Build()
	return
}

// TODO: works correct in Community Edition only. Have >1 VVM -> need to lock in a different way
func GetNextWSID(ctx context.Context, appStructs istructs.IAppStructs, clusterID istructs.ClusterID) (istructs.WSID, error) {
	vr := appStructs.ViewRecords()
	kb := vr.KeyBuilder(QNameViewNextBaseWSID)
	kb.PartitionKey().PutInt32(fldDummy1, 1)
	kb.ClusteringColumns().PutInt32(fldDummy2, 1) // no clustering columns -> storage driver behaviour is unknown
	nextBaseWSID := istructs.FirstBaseUserWSID
	nextWSIDGlobalLock.Lock()
	defer nextWSIDGlobalLock.Unlock()
	err := vr.Read(ctx, istructs.NullWSID, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		if nextBaseWSID != istructs.FirstBaseUserWSID {
			panic(">1 records in view NextBaseWSID")
		}
		nextBaseWSID = istructs.WSID(value.AsInt64(fldNextBaseWSID))
		return nil
	})
	if err != nil {
		return 0, err
	}
	vb := vr.NewValueBuilder(QNameViewNextBaseWSID)
	vb.PutInt64(fldNextBaseWSID, int64(nextBaseWSID+1))
	if err := vr.Put(istructs.NullWSID, kb, vb); err != nil {
		return 0, err
	}
	return istructs.NewWSID(clusterID, nextBaseWSID), nil
}
