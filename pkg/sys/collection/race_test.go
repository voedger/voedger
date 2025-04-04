/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
*/

package collection

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	cntItems = 20
)

type TSidsGeneratorType struct {
	lock           sync.Mutex
	idmap          map[istructs.RecordID]istructs.RecordID
	nextID         istructs.RecordID
	nextPlogOffset istructs.Offset
}

func (me *TSidsGeneratorType) NextID(tempID istructs.RecordID, _ appdef.IType) (storageID istructs.RecordID, err error) {
	me.lock.Lock()
	defer me.lock.Unlock()
	storageID = me.nextID
	me.nextID++
	me.idmap[tempID] = storageID
	return storageID, nil
}

func (me *TSidsGeneratorType) UpdateOnSync(_ istructs.RecordID, _ appdef.IType) {
	panic("must not be called")
}

func (me *TSidsGeneratorType) nextOffset() (offset istructs.Offset) {
	me.lock.Lock()
	defer me.lock.Unlock()
	offset = me.nextPlogOffset
	me.nextPlogOffset++
	return
}

func newTSIdsGenerator() *TSidsGeneratorType {
	return &TSidsGeneratorType{
		idmap:          make(map[istructs.RecordID]istructs.RecordID),
		nextID:         istructs.FirstBaseRecordID,
		nextPlogOffset: test.plogStartOfs,
	}
}

func Test_Race_SimpleInsertOne(t *testing.T) {
	req := require.New(t)

	_, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	idGen := newTSIdsGenerator()
	wg := sync.WaitGroup{}
	for i := 0; i < cntItems; i++ {
		wg.Add(1)
		go func(areq *require.Assertions, _ istructs.IAppStructs, aidGen *TSidsGeneratorType) {
			defer wg.Done()
			saveEvent(areq, appStructs, idGen, newTSModify(appStructs, aidGen, func(event istructs.IRawEventBuilder) {
				newDepartmentCUD(event, 1, 1, "Cold Drinks")
			}))
		}(req, appStructs, idGen)
	}
	wg.Wait()
}

func Test_Race_SimpleInsertMany(t *testing.T) {
	req := require.New(t)

	_, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	idGen := newTSIdsGenerator()
	wg := sync.WaitGroup{}
	for i := 0; i < cntItems; i++ {
		wg.Add(1)
		go func(areq *require.Assertions, _ istructs.IAppStructs, aidGen *TSidsGeneratorType, ai int) {
			defer wg.Done()
			saveEvent(areq, appStructs, idGen, newTSModify(appStructs, aidGen, func(event istructs.IRawEventBuilder) {
				newDepartmentCUD(event, istructs.RecordID(ai), int32(ai), "Hot Drinks"+strconv.Itoa(ai))
			}))
		}(req, appStructs, idGen, i+1)
	}
	wg.Wait()
}

func newTSModify(app istructs.IAppStructs, gen *TSidsGeneratorType, cb eventCallback) istructs.IRawEventBuilder {
	newOffset := gen.nextOffset()
	builder := app.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				HandlingPartition: test.partition,
				Workspace:         test.workspace,
				QName:             appdef.NewQName("test", "modify"),
				PLogOffset:        newOffset,
				WLogOffset:        newOffset,
			},
		})
	cb(builder)
	return builder
}
