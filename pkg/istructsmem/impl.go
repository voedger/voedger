/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	bytespool "github.com/valyala/bytebufferpool"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/descr"
	"github.com/voedger/voedger/pkg/istructsmem/internal/plogcache"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
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
	structures       map[appdef.AppQName]*appStructsType
	bucketsFactory   irates.BucketsFactoryType
	appTokensFactory payloads.IAppTokensFactory
	storageProvider  istorage.IAppStorageProvider
}

// istructs.IAppStructsProvider.BuiltIn
func (provider *appStructsProviderType) BuiltIn(appName appdef.AppQName) (structs istructs.IAppStructs, err error) {

	appCfg, ok := provider.configs[appName]
	if !ok {
		return nil, fmt.Errorf("%w: %v", istructs.ErrAppNotFound, appName)
	}

	provider.locker.Lock()
	defer provider.locker.Unlock()

	app, exists := provider.structures[appName]
	if !exists || !appCfg.Prepared() {
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

// istructs.IAppStructsProvider.New
func (provider *appStructsProviderType) New(name appdef.AppQName, def appdef.IAppDef, id istructs.ClusterAppID, wsCount istructs.NumAppWorkspaces) (istructs.IAppStructs, error) {
	provider.locker.Lock()
	defer provider.locker.Unlock()

	cfg := provider.configs.AddAppConfig(name, id, def, wsCount)
	buckets := provider.bucketsFactory()
	appTokens := provider.appTokensFactory.New(name)
	appStorage, err := provider.storageProvider.AppStorage(name)
	if err != nil {
		return nil, err
	}
	if err = cfg.prepare(buckets, appStorage); err != nil {
		return nil, err
	}
	app := newAppStructs(cfg, buckets, appTokens)
	//provider.structures[name] = app
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
	appTokens   istructs.IAppTokens
}

func newAppStructs(appCfg *AppConfigType, buckets irates.IBuckets, appTokens istructs.IAppTokens) *appStructsType {
	app := appStructsType{
		config:    appCfg,
		buckets:   buckets,
		appTokens: appTokens,
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

// istructs.IAppStructs.ObjectBuilder
func (app *appStructsType) ObjectBuilder(name appdef.QName) istructs.IObjectBuilder {
	return newObject(app.config, name, nil)
}

// istructs.IAppStructs.Resources
func (app *appStructsType) Resources() istructs.IResources {
	return &app.config.Resources
}

// istructs.IAppStructs.ClusterAppID
func (app *appStructsType) ClusterAppID() istructs.ClusterAppID {
	return app.config.ClusterAppID
}

// istructs.IAppStructs.AppQName
func (app *appStructsType) AppQName() appdef.AppQName {
	return app.config.Name
}

func (app *appStructsType) AppTokens() istructs.IAppTokens {
	return app.appTokens
}

// istructs.IAppStructs.AsyncProjectors
func (app *appStructsType) AsyncProjectors() istructs.Projectors {
	return app.config.AsyncProjectors()
}

// IAppStructs.Buckets() - wrong, import cycle between istructs and irates
// so let's add simple method to use it at utils.IBucketsFromIAppStructs()
func (app *appStructsType) Buckets() irates.IBuckets {
	return app.buckets
}

func (app *appStructsType) CUDValidators() []istructs.CUDValidator {
	return app.config.cudValidators
}

func (app *appStructsType) EventValidators() []istructs.EventValidator {
	return app.config.eventValidators
}

// func (app appStructsType) GetAppStorage

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

// istructs.IAppStructs.SyncProjectors
func (app *appStructsType) SyncProjectors() istructs.Projectors {
	return app.config.SyncProjectors()
}

func (app *appStructsType) NumAppWorkspaces() istructs.NumAppWorkspaces {
	return app.config.numAppWorkspaces
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
	app       *appStructsType
	plogCache *plogcache.Cache
}

func newEvents(app *appStructsType) appEventsType {
	return appEventsType{
		app:       app,
		plogCache: plogcache.New(app.config.Params.PLogEventCacheSize),
	}
}

// istructs.IEvents.BuildPLogEvent
func (e *appEventsType) BuildPLogEvent(ev istructs.IRawEvent) istructs.IPLogEvent {
	dbEvent := ev.(*eventType)

	if n := dbEvent.QName(); n != istructs.QNameForCorruptedData {
		panic(fmt.Errorf("%w: QName() is «%v», expected «%v»", ErrorEventNotValid, n, istructs.QNameForCorruptedData))
	}

	if o := dbEvent.PLogOffset(); o != istructs.NullOffset {
		panic(fmt.Errorf("%w: PLogOffset() is «%v», expected «%v»", ErrorEventNotValid, o, istructs.NullOffset))
	}

	return dbEvent
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
func (e *appEventsType) PutPlog(ev istructs.IRawEvent, buildErr error, generator istructs.IIDGenerator) (event istructs.IPLogEvent, err error) {
	dbEvent := ev.(*eventType)

	if buildErr != nil {
		dbEvent.setBuildError(buildErr)
	}

	if dbEvent.valid() {
		if err := dbEvent.regenerateIDs(generator); err != nil {
			dbEvent.setBuildError(err)
		}
	}

	if dbEvent.argUnlObj.QName() != appdef.NullQName {
		dbEvent.argUnlObj.maskValues()
	}

	p, o := ev.HandlingPartition(), ev.PLogOffset()
	pKey, cCols := plogKey(p, o)

	evData := dbEvent.storeToBytes()

	if err = e.app.config.storage.Put(pKey, cCols, evData); err == nil {
		event = dbEvent
		e.plogCache.Put(p, o, event)
	}

	return event, err
}

// istructs.IEvents.PutWlog
func (e *appEventsType) PutWlog(ev istructs.IPLogEvent) (err error) {
	pKey, cCols := wlogKey(ev.Workspace(), ev.WLogOffset())
	evData := ev.(*eventType).storeToBytes()

	return e.app.config.storage.Put(pKey, cCols, evData)
}

// istructs.IEvents.ReadPLog
func (e *appEventsType) ReadPLog(ctx context.Context, partition istructs.PartitionID, offset istructs.Offset, toReadCount int, cb istructs.PLogEventsReaderCallback) error {

	switch toReadCount {
	case 1:
		// See [#292](https://github.com/voedger/voedger/issues/292)
		if e, ok := e.plogCache.Get(partition, offset); ok {
			return cb(offset, e)
		}

		pKey, cCols := plogKey(partition, offset)
		data := bytespool.Get()
		ok, err := e.app.config.storage.Get(pKey, cCols, &data.B)
		if ok {
			event := newEvent(e.app.config)
			if err = event.loadFromBytes(data.B); err == nil {
				event.buffer = data
				err = cb(offset, event)
			}
		} else {
			bytespool.Put(data)
		}
		return err
	default:
		return readLogParts(offset, toReadCount, func(ofsHi uint64, ofsLo1, ofsLo2 uint16) (ok bool, err error) {
			count := 0
			pKey, cFrom := plogKey(partition, glueLogOffset(ofsHi, ofsLo1))
			cTo := uint16bytes(ofsLo2 + 1) // storage.Read() pass half-open interval [cFrom, cTo)
			if ofsLo2 >= lowMask {
				cTo = nil
			}
			err = e.app.config.storage.Read(ctx, pKey, cFrom, cTo, func(ccols, data []byte) error {
				count++
				ofs := glueLogOffset(ofsHi, binary.BigEndian.Uint16(ccols))
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
		pKey, cCols := wlogKey(workspace, offset)
		data := bytespool.Get()
		ok, err := e.app.config.storage.Get(pKey, cCols, &data.B)
		if ok {
			event := newEvent(e.app.config)
			if err = event.loadFromBytes(data.B); err == nil {
				event.buffer = data
				err = cb(offset, event)
			}
		} else {
			bytespool.Put(data)
		}
		return err
	default:
		return readLogParts(offset, toReadCount, func(ofsHi uint64, ofsLo1, ofsLo2 uint16) (ok bool, err error) {
			count := 0
			pKey, cFrom := wlogKey(workspace, glueLogOffset(ofsHi, ofsLo1))
			cTo := uint16bytes(ofsLo2 + 1) // storage.Read() pass half-open interval [cFrom, cTo)
			if ofsLo2 >= lowMask {
				cTo = nil
			}
			err = e.app.config.storage.Read(ctx, pKey, cFrom, cTo, func(ccols, data []byte) error {
				count++
				ofs := glueLogOffset(ofsHi, binary.BigEndian.Uint16(ccols))
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
	pk, cc := recordKey(workspace, id)
	return recs.app.config.storage.Get(pk, cc, data)
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
		pk, cc := recordKey(workspace, ids[i].ID)
		batch, ok := plan[string(pk)]
		if !ok {
			batch = make([]istorage.GetBatchItem, 0, len(ids)-i) // to prevent reallocation
		}
		batch = append(batch, istorage.GetBatchItem{CCols: cc, Data: new([]byte)})
		plan[string(pk)] = batch
		batches[i] = &batch[len(batch)-1]
	}
	for idHi, batch := range plan {
		if err = recs.app.config.storage.GetBatch([]byte(idHi), batch); err != nil {
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
			ids[i].Record = rec
		}
	}
	return nil
}

// putRecord puts record to application storage through view-records methods
func (recs *appRecordsType) putRecord(workspace istructs.WSID, id istructs.RecordID, data []byte) (err error) {
	pk, cc := recordKey(workspace, id)
	return recs.app.config.storage.Put(pk, cc, data)
}

// putRecordsBatch puts record array to application storage through view-records batch methods
type recordBatchItemType struct {
	id   istructs.RecordID
	data []byte
}

func (recs *appRecordsType) putRecordsBatch(workspace istructs.WSID, records []recordBatchItemType) (err error) {
	batch := make([]istorage.BatchItem, len(records))
	for i, r := range records {
		batch[i].PKey, batch[i].CCols = recordKey(workspace, r.id)
		batch[i].Value = r.data
	}
	return recs.app.config.storage.PutBatch(batch)
}

// validEvent returns error if event has non-committable data, such as singleton unique violations or non exists updated record id
func (recs *appRecordsType) validEvent(ev *eventType) (err error) {

	load := func(id istructs.RecordID, rec *recordType) (exists bool, err error) {
		data := make([]byte, 0)
		if exists, err = recs.getRecord(ev.ws, id, &data); exists {
			if rec != nil {
				err = rec.loadFromBytes(data)
			}
		}

		return exists, err
	}

	for _, rec := range ev.cud.creates {
		if singleton, ok := rec.typ.(appdef.ISingleton); ok && singleton.Singleton() {
			id, err := recs.app.config.singletons.ID(rec.QName())
			if err != nil {
				return err
			}
			exists, err := load(id, nil)
			if err != nil {
				return fmt.Errorf("error checking singleton «%v» record «%d» existence: %w", rec.QName(), id, err)
			}
			if exists {
				return fmt.Errorf("can not create singleton, «%v» record «%d» already exists: %w", rec.QName(), id, ErrRecordIDUniqueViolation)
			}
		}
	}

	for _, rec := range ev.cud.updates {
		old := newRecord(recs.app.config)
		exists, err := load(rec.originRec.ID(), old)
		if err != nil {
			return fmt.Errorf("error load updated «%v» record «%d»: %w", rec.originRec.QName(), rec.originRec.ID(), err)
		}
		if !exists {
			return fmt.Errorf("updated «%v» record «%d» not exists: %w", rec.originRec.QName(), rec.originRec.ID(), ErrRecordIDNotFound)
		}

		// check exists record has correct QName
		if rec.originRec.QName() != old.QName() {
			return fmt.Errorf("updated «%v» record «%d» has unexpected QName value «%v»: %w", rec.originRec.QName(), rec.originRec.ID(), old.QName(), ErrWrongType)
		}
	}

	return nil
}

// istructs.IRecords.Apply
func (recs *appRecordsType) Apply(event istructs.IPLogEvent) (err error) {
	return recs.Apply2(event, nil)
}

// istructs.IRecords.Apply2
func (recs *appRecordsType) Apply2(event istructs.IPLogEvent, cb func(rec istructs.IRecord)) (err error) {
	ev := event.(*eventType)

	if !ev.Error().ValidEvent() {
		panic(fmt.Errorf("can not apply not valid event: %s: %w", ev.Error().ErrStr(), ErrorEventNotValid))
	}

	records := make([]*recordType, 0)
	batch := make([]recordBatchItemType, 0)

	store := func(rec *recordType) error {
		data := rec.storeToBytes()
		records = append(records, rec)
		batch = append(batch, recordBatchItemType{rec.ID(), data})

		return nil
	}

	load := func(rec *recordType) error {
		data := make([]byte, 0)
		exists, err := recs.getRecord(ev.ws, rec.ID(), &data)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("record «%d» not exists: %w", rec.ID(), ErrRecordIDNotFound)
		}
		return rec.loadFromBytes(data)
	}

	if err = ev.cud.applyRecs(load, store); err == nil {
		if len(records) > 0 {
			if err = recs.putRecordsBatch(ev.ws, batch); err == nil {
				if cb != nil {
					for _, rec := range records {
						cb(rec)
					}
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
			return rec, nil
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

func (recs *appRecordsType) GetSingletonID(qName appdef.QName) (istructs.RecordID, error) {
	return recs.app.config.singletons.ID(qName)
}

// istructs.IRecords.PutJSON
func (recs *appRecordsType) PutJSON(ws istructs.WSID, j map[appdef.FieldName]any) error {
	rec := newRecord(recs.app.config)

	rec.PutFromJSON(j)

	if err := rec.build(); err != nil {
		return err
	}

	storable := func(k appdef.TypeKind) bool {
		return k == appdef.TypeKind_GDoc || k == appdef.TypeKind_CDoc || k == appdef.TypeKind_WDoc ||
			k == appdef.TypeKind_GRecord || k == appdef.TypeKind_CRecord || k == appdef.TypeKind_WRecord
	}

	if k := rec.typeDef().Kind(); !storable(k) {
		return fmt.Errorf("%v is not storable record type: %w", rec.typeDef(), ErrWrongType)
	}

	if rec.ID() == istructs.NullRecordID {
		return fmt.Errorf("can not put record with null %s: %w", appdef.SystemField_ID, ErrFieldIsEmpty)
	}
	if rec.ID().IsRaw() {
		return fmt.Errorf("can not put record with raw %s: %w", appdef.SystemField_ID, ErrRawRecordIDUnexpected)
	}

	return recs.putRecord(ws, rec.ID(), rec.storeToBytes())
}
