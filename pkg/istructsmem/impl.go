/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"

	"github.com/voedger/voedger/pkg/istructsmem/internal/bytespool"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/descr"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

// appStructsProviderType implements IAppStructsProvider interface
//   - fields:
//     — locker: to implement @ConcurrentAccess-methods
//     — configs: configurations of supported applications
//     — structures: maps of application structures
//   - methods:
//   - interfaces:
//     — istructs.IAppStructsProvider
type appStructsProviderType struct {
	locker           sync.RWMutex
	configs          AppConfigsType
	structures       map[istructs.AppQName]*appStructsType
	bucketsFactory   irates.BucketsFactoryType
	appTokensFactory payloads.IAppTokensFactory
	storageProvider  istorage.IAppStorageProvider
}

// istructs.IAppStructsProvider.AppStructs
func (provider *appStructsProviderType) AppStructs(appName istructs.AppQName) (structs istructs.IAppStructs, err error) {

	appCfg, ok := provider.configs[appName]
	if !ok {
		return nil, istructs.ErrAppNotFound
	}

	provider.locker.Lock()
	defer provider.locker.Unlock()

	app, exists := provider.structures[appName]
	if !exists {
		buckets := provider.bucketsFactory()
		appTokens := provider.appTokensFactory.New(appName)
		appStorage, err := provider.storageProvider.AppStorage(appName)
		if err != nil {
			return nil, err
		}
		if err = appCfg.prepare(buckets, appStorage); err != nil {
			return nil, err
		}
		app = newAppStructs(appCfg, buckets, appTokens)
		provider.structures[appName] = app
	}
	return app, nil
}

// appStructsType implements IAppStructs interface
//   - interfaces:
//     — istructs.IAppStructs
type appStructsType struct {
	config      *AppConfigType
	events      appEventsType
	records     appRecordsType
	viewRecords appViewRecords
	buckets     irates.IBuckets
	descr       *descr.Application
	appWSAmount istructs.AppWSAmount
	appTokens   istructs.IAppTokens
}

func newAppStructs(appCfg *AppConfigType, buckets irates.IBuckets, appTokens istructs.IAppTokens) *appStructsType {
	app := appStructsType{
		config:      appCfg,
		buckets:     buckets,
		appWSAmount: istructs.DefaultAppWSAmount,
		appTokens:   appTokens,
	}
	app.events = newEvents(&app)
	app.records = newRecords(&app)
	app.viewRecords = newAppViewRecords(&app)
	appCfg.app = &app
	return &app
}

// istructs.IAppStructs.AppDef
func (app *appStructsType) AppDef() appdef.IAppDef {
	return app.config.AppDef
}

// istructs.IAppStructs.Events
func (app *appStructsType) Events() istructs.IEvents {
	return &app.events
}

// istructs.IAppStructs.Records
func (app *appStructsType) Records() istructs.IRecords {
	return &app.records
}

// istructs.IAppStructs.ViewRecords
func (app *appStructsType) ViewRecords() istructs.IViewRecords {
	return &app.viewRecords
}

// istructs.IAppStructs.Resources
func (app *appStructsType) Resources() istructs.IResources {
	return &app.config.Resources
}

// istructs.IAppStructs.ClusterAppID
func (app *appStructsType) ClusterAppID() istructs.ClusterAppID {
	return app.config.QNameID
}

// istructs.IAppStructs.AppQName
func (app *appStructsType) AppQName() istructs.AppQName {
	return app.config.Name
}

// IAppStructs.Buckets() - wrong, import cycle between istructs and irates
// so let's add simple method to use it at utils.IBucketsFromIAppStructs()
func (app *appStructsType) Buckets() irates.IBuckets {
	return app.buckets
}

func (app *appStructsType) SyncProjectors() []istructs.ProjectorFactory {
	return app.config.syncProjectorFactories
}
func (app *appStructsType) AsyncProjectors() []istructs.ProjectorFactory {
	return app.config.asyncProjectorFactories
}

func (app *appStructsType) CUDValidators() []istructs.CUDValidator {
	return app.config.cudValidators
}

func (app *appStructsType) EventValidators() []istructs.EventValidator {
	return app.config.eventValidators
}

func (app *appStructsType) WSAmount() istructs.AppWSAmount {
	return app.appWSAmount
}

func (app *appStructsType) AppTokens() istructs.IAppTokens {
	return app.appTokens
}

func (app *appStructsType) IsFunctionRateLimitsExceeded(funcQName appdef.QName, wsid istructs.WSID) bool {
	rateLimits, ok := app.config.FunctionRateLimits.limits[funcQName]
	if !ok {
		return false
	}
	keys := []irates.BucketKey{}
	for rlKind := range rateLimits {
		key := irates.BucketKey{
			QName:         funcQName,
			RateLimitName: GetFunctionRateLimitName(funcQName, rlKind),
		}
		// already checked for unsupported kind on appStructs.prepare() stage
		switch rlKind {
		case istructs.RateLimitKind_byApp:
			key.App = app.config.Name
		case istructs.RateLimitKind_byWorkspace:
			key.Workspace = wsid
		case istructs.RateLimitKind_byID:
			// skip
			continue
		}
		keys = append(keys, key)
	}
	return !app.buckets.TakeTokens(keys, 1)
}

func (app *appStructsType) describe() *descr.Application {
	if app.descr == nil {
		app.descr = descr.Provide(app, app.config.FunctionRateLimits.limits)
	}
	return app.descr
}

// istructs.IAppStructs.DescribePackageNames: Describe package names
func (app *appStructsType) DescribePackageNames() (names []string) {
	for name := range app.describe().Packages {
		names = append(names, name)
	}
	return names
}

// istructs.IAppStructs.DescribePackage: Describe package content
func (app *appStructsType) DescribePackage(name string) interface{} {
	return app.describe().Packages[name]
}

// appEventsType implements IEvents
//   - interfaces:
//     — istructs.IEvents
type appEventsType struct {
	app *appStructsType
}

func newEvents(app *appStructsType) appEventsType {
	return appEventsType{
		app: app,
	}
}

// istructs.IEvents.GetSyncRawEventBuilder
func (e *appEventsType) GetSyncRawEventBuilder(params istructs.SyncRawEventBuilderParams) istructs.IRawEventBuilder {
	return newSyncEventBuilder(e.app.config, params)
}

// istructs.IEvents.GetNewRawEventBuilder
func (e *appEventsType) GetNewRawEventBuilder(params istructs.NewRawEventBuilderParams) istructs.IRawEventBuilder {
	return newEventBuilder(e.app.config, params)
}

// istructs.IEvents.PutPlog
func (e *appEventsType) PutPlog(ev istructs.IRawEvent, buildErr error, generator istructs.IDGenerator) (event istructs.IPLogEvent, err error) {
	dbEvent := ev.(*eventType)

	dbEvent.setBuildError(buildErr)
	if dbEvent.valid() {
		if err := dbEvent.regenerateIDs(generator); err != nil {
			dbEvent.setBuildError(err)
		}
	}

	if dbEvent.argUnlObj.QName() != appdef.NullQName {
		dbEvent.argUnlObj.maskValues()
	}

	evData := dbEvent.storeToBytes()
	pKey, cCols := splitLogOffset(ev.PLogOffset())
	pKey = utils.PrefixBytes(pKey, consts.SysView_PLog, ev.HandlingPartition()) // + partition! see #18047

	if err = e.app.config.storage.Put(pKey, cCols, evData); err == nil {
		event = dbEvent
	}

	return event, err
}

// istructs.IEvents.PutWlog
func (e *appEventsType) PutWlog(ev istructs.IPLogEvent) (err error) {
	evData := ev.(*eventType).storeToBytes()
	pKey, cCols := splitLogOffset(ev.WLogOffset())
	pKey = utils.PrefixBytes(pKey, consts.SysView_WLog, ev.Workspace())

	return e.app.config.storage.Put(pKey, cCols, evData)
}

// istructs.IEvents.ReadPLog
func (e *appEventsType) ReadPLog(ctx context.Context, partition istructs.PartitionID, offset istructs.Offset, toReadCount int, cb istructs.PLogEventsReaderCallback) error {

	switch toReadCount {
	case 1:
		// See [#292](https://github.com/voedger/voedger/issues/292)
		pKey, cCol := splitLogOffset(offset)
		pKey = utils.PrefixBytes(pKey, consts.SysView_PLog, partition) // + partition! see #18047
		data := bytespool.Get()
		ok, err := e.app.config.storage.Get(pKey, cCol, &data)
		if ok {
			event := newEvent(e.app.config)
			if err = event.loadFromBytes(data); err == nil {
				event.pooledBytes = data
				err = cb(offset, event)
			}
		} else {
			bytespool.Put(data)
		}
		return err
	default:
		return readLogParts(offset, toReadCount, func(pk, ccFrom, ccTo []byte) (ok bool, err error) {
			count := 0
			pKey := utils.PrefixBytes(pk, consts.SysView_PLog, partition) // + partition! see #18047
			err = e.app.config.storage.Read(ctx, pKey, ccFrom, ccTo, func(ccols, data []byte) error {
				count++
				ofs := calcLogOffset(pk, ccols)
				event := newEvent(e.app.config)
				if err = event.loadFromBytes(data); err == nil {
					err = cb(ofs, event)
				}
				return err
			})
			return (err == nil) && (count > 0), err // stop iterate parts if error or no events in last partition
		})
	}
}

// istructs.IEvents.ReadWLog
func (e *appEventsType) ReadWLog(ctx context.Context, workspace istructs.WSID, offset istructs.Offset, toReadCount int, cb istructs.WLogEventsReaderCallback) error {

	switch toReadCount {
	case 1:
		// See [#292](https://github.com/voedger/voedger/issues/292)
		pKey, cCol := splitLogOffset(offset)
		pKey = utils.PrefixBytes(pKey, consts.SysView_WLog, workspace)
		data := bytespool.Get()
		ok, err := e.app.config.storage.Get(pKey, cCol, &data)
		if ok {
			event := newEvent(e.app.config)
			if err = event.loadFromBytes(data); err == nil {
				event.pooledBytes = data
				err = cb(offset, event)
			}
		} else {
			bytespool.Put(data)
		}
		return err
	default:
		return readLogParts(offset, toReadCount, func(pk, ccFrom, ccTo []byte) (ok bool, err error) {
			count := 0
			pKey := utils.PrefixBytes(pk, consts.SysView_WLog, workspace)
			err = e.app.config.storage.Read(ctx, pKey, ccFrom, ccTo, func(ccols, data []byte) error {
				count++
				ofs := calcLogOffset(pk, ccols)
				event := newEvent(e.app.config)
				if err = event.loadFromBytes(data); err == nil {
					err = cb(ofs, event)
				}
				return err
			})
			return (err == nil) && (count > 0), err // stop iterate parts if error or no events in last partition
		})
	}
}

// appRecordsType implements IRecords
//   - interfaces:
//     — istructs.IRecords
type appRecordsType struct {
	app *appStructsType
}

func newRecords(app *appStructsType) appRecordsType {
	return appRecordsType{
		app: app,
	}
}

// getRecord reads record from application storage through view-records methods
func (recs *appRecordsType) getRecord(workspace istructs.WSID, id istructs.RecordID, data *[]byte) (ok bool, err error) {
	idHi, idLow := splitRecordID(id)
	pk := utils.PrefixBytes(idHi, consts.SysView_Records, workspace)
	return recs.app.config.storage.Get(pk, idLow, data)
}

// getRecordBatch reads record from application storage through view-records methods
func (recs *appRecordsType) getRecordBatch(workspace istructs.WSID, ids []istructs.RecordGetBatchItem) (err error) {
	if len(ids) > maxGetBatchRecordCount {
		return fmt.Errorf("batch read %d records requested, but only %d supported: %w", len(ids), maxGetBatchRecordCount, ErrMaxGetBatchRecordCountExceeds)
	}
	batches := make([]*istorage.GetBatchItem, len(ids))
	plan := make(map[string][]istorage.GetBatchItem)
	for i := 0; i < len(ids); i++ {
		ids[i].Record = NewNullRecord(ids[i].ID)
		idHi, idLow := splitRecordID(ids[i].ID)
		batch, ok := plan[string(idHi)]
		if !ok {
			batch = make([]istorage.GetBatchItem, 0, len(ids)) // to prevent reallocation
		}
		batch = append(batch, istorage.GetBatchItem{CCols: idLow, Data: new([]byte)})
		plan[string(idHi)] = batch
		batches[i] = &batch[len(batch)-1]
	}
	for idHi, batch := range plan {
		pk := utils.PrefixBytes([]byte(idHi), consts.SysView_Records, workspace)
		if err = recs.app.config.storage.GetBatch(pk, batch); err != nil {
			return err
		}
	}
	for i := 0; i < len(batches); i++ {
		b := batches[i]
		if b.Ok {
			rec := newRecord(recs.app.config)
			if err = rec.loadFromBytes(*b.Data); err != nil {
				return err
			}
			ids[i].Record = &rec
		}
	}
	return nil
}

// putRecord puts record to application storage through view-records methods
func (recs *appRecordsType) putRecord(workspace istructs.WSID, id istructs.RecordID, data []byte) (err error) {
	idHi, idLow := splitRecordID(id)
	pk := utils.PrefixBytes(idHi, consts.SysView_Records, workspace)
	return recs.app.config.storage.Put(pk, idLow, data)
}

// putRecordsBatch puts record array to application storage through view-records batch methods
type recordBatchItemType struct {
	id   istructs.RecordID
	data []byte
}

func (recs *appRecordsType) putRecordsBatch(workspace istructs.WSID, records []recordBatchItemType) (err error) {
	batch := make([]istorage.BatchItem, len(records))
	for i, r := range records {
		idHi, idLow := splitRecordID(r.id)
		batch[i].PKey = utils.PrefixBytes(idHi, consts.SysView_Records, workspace)
		batch[i].CCols = idLow
		batch[i].Value = r.data
	}
	return recs.app.config.storage.PutBatch(batch)
}

// validEvent returns error if event has non-committable data, such as singleton unique violations or invalid record id references
func (recs *appRecordsType) validEvent(ev *eventType) (err error) {

	existsRecord := func(id istructs.RecordID) (bool, error) {
		data := make([]byte, 0)
		ok, err := recs.getRecord(ev.ws, id, &data)
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	for _, rec := range ev.cud.creates {
		if cDoc, ok := rec.def.(appdef.ICDoc); ok && cDoc.Singleton() {
			id, err := recs.app.config.singletons.ID(rec.QName())
			if err != nil {
				return err
			}
			isExists, err := existsRecord(id)
			if err != nil {
				return err
			}
			if isExists {
				return fmt.Errorf("can not create singleton, CDoc «%v» record «%d» already exists: %w", rec.QName(), id, ErrRecordIDUniqueViolation)
			}
		}
	}

	return nil
}

// istructs.IRecords.Apply
func (recs *appRecordsType) Apply(event istructs.IPLogEvent) (err error) {
	return recs.Apply2(event, func(istructs.IRecord) {})
}

// istructs.IRecords.Apply2
func (recs *appRecordsType) Apply2(event istructs.IPLogEvent, cb func(rec istructs.IRecord)) (err error) {
	ev := event.(*eventType)

	if !ev.Error().ValidEvent() {
		panic(fmt.Errorf("can not apply not valid event: %s: %w", ev.Error().ErrStr(), ErrorEventNotValid))
	}

	existsRecord := func(id istructs.RecordID) (bool, error) {
		data := make([]byte, 0)
		ok, err := recs.getRecord(ev.ws, id, &data)
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	loadRecord := func(rec *recordType) error {
		data := make([]byte, 0)
		var ok bool
		if ok, err = recs.getRecord(ev.ws, rec.ID(), &data); ok {
			err = rec.loadFromBytes(data)
		}

		return err
	}

	records := make([]*recordType, 0)
	batch := make([]recordBatchItemType, 0)

	storeRecord := func(rec *recordType) error {
		data := rec.storeToBytes()
		records = append(records, rec)
		batch = append(batch, recordBatchItemType{rec.ID(), data})

		return nil
	}

	if err = ev.applyCommandRecs(existsRecord, loadRecord, storeRecord); err == nil {
		if len(records) > 0 {
			if err = recs.putRecordsBatch(ev.ws, batch); err == nil {
				for _, rec := range records {
					cb(rec)
				}
			}
		}
	}

	return err
}

// istructs.IRecords.Get
func (recs *appRecordsType) Get(workspace istructs.WSID, highConsistency bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	data := make([]byte, 0)
	var ok bool
	if ok, err = recs.getRecord(workspace, id, &data); ok {
		rec := newRecord(recs.app.config)
		if err = rec.loadFromBytes(data); err == nil {
			return &rec, nil
		}
	}
	return NewNullRecord(id), err
}

// istructs.IRecords.GetBatch
func (recs *appRecordsType) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
	return recs.getRecordBatch(workspace, ids)
}

// istructs.IRecords.GetSingleton
func (recs *appRecordsType) GetSingleton(workspace istructs.WSID, qName appdef.QName) (record istructs.IRecord, err error) {
	var id istructs.RecordID
	if id, err = recs.app.config.singletons.ID(qName); err != nil {
		return NewNullRecord(istructs.NullRecordID), err
	}
	return recs.Get(workspace, true, id)
}
