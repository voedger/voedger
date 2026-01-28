/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/binary"
	"strconv"
	"sync"

	bytespool "github.com/valyala/bytebufferpool"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/descr"
	"github.com/voedger/voedger/pkg/istructsmem/internal/plogcache"
	"github.com/voedger/voedger/pkg/istructsmem/internal/recreg"
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
	locker               sync.RWMutex
	configs              AppConfigsType
	structures           map[appdef.AppQName]*appStructsType
	bucketsFactory       irates.BucketsFactoryType
	appTokensFactory     payloads.IAppTokensFactory
	storageProvider      istorage.IAppStorageProvider
	seqTrustLevel        isequencer.SequencesTrustLevel
	appTTLStorageFactory istructs.AppTTLStorageFactory
}

// istructs.IAppStructsProvider.BuiltIn
func (provider *appStructsProviderType) BuiltIn(appName appdef.AppQName) (structs istructs.IAppStructs, err error) {

	appCfg, ok := provider.configs[appName]
	if !ok {
		return nil, enrichError(istructs.ErrAppNotFound, appName)
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
		var appTTLStorage istructs.IAppTTLStorage
		if provider.appTTLStorageFactory != nil {
			appTTLStorage = provider.appTTLStorageFactory(appCfg.ClusterAppID)
		}
		app = newAppStructs(appCfg, buckets, appTokens, provider.seqTrustLevel, appTTLStorage)
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
	var appTTLStorage istructs.IAppTTLStorage
	if provider.appTTLStorageFactory != nil {
		appTTLStorage = provider.appTTLStorageFactory(id)
	}
	app := newAppStructs(cfg, buckets, appTokens, provider.seqTrustLevel, appTTLStorage)
	//provider.structures[name] = app
	return app, nil
}

// appStructsType implements IAppStructs interface
//   - interfaces:
//     — istructs.IAppStructs
type appStructsType struct {
	config        *AppConfigType
	events        appEventsType
	records       appRecordsType
	viewRecords   appViewRecords
	buckets       irates.IBuckets
	descr         *descr.Application
	appTokens     istructs.IAppTokens
	seqTrustLevel isequencer.SequencesTrustLevel
	appTTLStorage istructs.IAppTTLStorage
}

func newAppStructs(appCfg *AppConfigType, buckets irates.IBuckets, appTokens istructs.IAppTokens, seqTrustLevel isequencer.SequencesTrustLevel, appTTLStorage istructs.IAppTTLStorage) *appStructsType {
	app := appStructsType{
		config:        appCfg,
		buckets:       buckets,
		appTokens:     appTokens,
		seqTrustLevel: seqTrustLevel,
		appTTLStorage: appTTLStorage,
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

// istructs.IAppStructs.AppTTLStorage
func (app *appStructsType) AppTTLStorage() istructs.IAppTTLStorage {
	return app.appTTLStorage
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

func (app *appStructsType) SeqTypes() map[istructs.QNameID]map[istructs.QNameID]uint64 {
	res := map[istructs.QNameID]map[istructs.QNameID]uint64{}
	for wsKind, seqTypes := range app.config.seqTypes {
		wsKindSeqTypes, ok := res[istructs.QNameID(wsKind)]
		if !ok {
			wsKindSeqTypes = map[istructs.QNameID]uint64{}
			res[istructs.QNameID(wsKind)] = wsKindSeqTypes
		}
		for seqID, number := range seqTypes {
			wsKindSeqTypes[istructs.QNameID(seqID)] = uint64(number)
		}
	}
	return res
}

func (app *appStructsType) QNameID(qName appdef.QName) (istructs.QNameID, error) {
	return app.config.QNameID(qName)
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
	ok, _ = app.buckets.TakeTokens(keys, 1)
	return !ok
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
		app.descr = descr.Provide(app.AppQName(), app.AppDef())
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
func (app *appStructsType) DescribePackage(name string) any {
	return app.describe().Packages[name]
}

// istructs.IAppStructs.GetEventReapplier
func (app *appStructsType) GetEventReapplier(plogEvent istructs.IPLogEvent) istructs.IEventReapplier {
	if !plogEvent.(*eventType).isStored {
		panic("only events read from the storage can be re-applied")
	}
	return &implIEventReapplier{
		plogEvent: plogEvent,
		app:       app,
	}
}

type implIEventReapplier struct {
	plogEvent istructs.IPLogEvent
	app       *appStructsType
}

func (er *implIEventReapplier) PutWLog() error {
	pKey, cCols, data := getEventBytes(er.plogEvent)
	return er.app.config.storage.Put(pKey, cCols, data)
}

func (er *implIEventReapplier) ApplyRecords() error {
	return er.app.records.apply2(er.plogEvent, nil, true)
}

// appEventsType implements IEvents
//   - interfaces:
//     — istructs.IEvents
type appEventsType struct {
	app       *appStructsType
	plogCache *plogcache.Cache
	recsReg   *recreg.Registry
}

func newEvents(app *appStructsType) appEventsType {
	return appEventsType{
		app:       app,
		plogCache: plogcache.New(app.config.Params.PLogEventCacheSize),
		recsReg:   recreg.New(func() istructs.IViewRecords { return &app.viewRecords }),
	}
}

// istructs.IEvents.BuildPLogEvent
func (e *appEventsType) BuildPLogEvent(ev istructs.IRawEvent) istructs.IPLogEvent {
	dbEvent := ev.(*eventType)

	if n := dbEvent.QName(); n != istructs.QNameForCorruptedData {
		panic(ErrorEventNotValid("QName() is «%v», expected «%v»", n, istructs.QNameForCorruptedData))
	}

	if o := dbEvent.PLogOffset(); o != istructs.NullOffset {
		panic(ErrorEventNotValid("PLogOffset() is «%v», expected «%v»", o, istructs.NullOffset))
	}

	return dbEvent
}

// istructs.IEvents.FindORec #3711 ~impl~
func (e *appEventsType) FindORec(workspace istructs.WSID, id istructs.RecordID) (istructs.Offset, error) {
	qn, ofs, err := e.recsReg.Get(workspace, id)
	if err != nil {
		// Failed get from record registry, enriched error should be returned
		return istructs.NullOffset, enrichError(err, "ws %d, record id %d", workspace, id)
	}
	if (qn != appdef.NullQName) && (ofs != istructs.NullOffset) {
		if kind := e.app.AppDef().Type(qn).Kind(); recordsInWLog.Contains(kind) {
			// Successfully founded
			return ofs, nil
		}
	}
	// ORecord is not founded in records registry
	return istructs.NullOffset, nil
}

// istructs.IEvents.GetNewRawEventBuilder
func (e *appEventsType) GetNewRawEventBuilder(params istructs.NewRawEventBuilderParams) istructs.IRawEventBuilder {
	return newEventBuilder(e.app.config, params)
}

// istructs.IEvents.GetORec #3711 ~impl~
func (e *appEventsType) GetORec(workspace istructs.WSID, id istructs.RecordID, wlog istructs.Offset) (istructs.IRecord, error) {
	if wlog == istructs.NullOffset {
		ofs, err := e.FindORec(workspace, id)
		if err != nil {
			// Failed find in record registry
			return NewNullRecord(id), err
		}
		if ofs == istructs.NullOffset {
			// ORecord is not founded in records registry
			return NewNullRecord(id), nil
		}
		wlog = ofs
	}

	record, found := NewNullRecord(id), false
	err := e.app.events.ReadWLog(context.Background(), workspace, wlog, 1, func(_ istructs.Offset, event istructs.IWLogEvent) error {
		if o, ok := event.ArgumentObject().(*objectType); ok {
			if o := o.find(func(o *objectType) bool { return o.ID() == id }); o != nil {
				// found record with id within event argument structure
				dupe := newRecord(e.app.config)
				dupe.copyFrom(&o.recordType)
				record = dupe
				found = true
			}
		}
		return nil
	})
	if err != nil {
		// Failed wlog reading, enriched error should be returned
		return NewNullRecord(id), enrichError(err, "ws %d, wlog offset %d, record id %d", err, workspace, wlog, id)
	}
	if !found {
		// Offset founded, but event argument has no record with requested id
		return NewNullRecord(id), nil
	}

	return record, nil
}

// istructs.IEvents.GetSyncRawEventBuilder
func (e *appEventsType) GetSyncRawEventBuilder(params istructs.SyncRawEventBuilderParams) istructs.IRawEventBuilder {
	return newSyncEventBuilder(e.app.config, params)
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

	switch {
	case dbEvent.name == istructs.QNameForCorruptedData, e.app.seqTrustLevel == isequencer.SequencesTrustLevel_2:
		err = e.app.config.storage.Put(pKey, cCols, evData)
	case e.app.seqTrustLevel == isequencer.SequencesTrustLevel_0, e.app.seqTrustLevel == isequencer.SequencesTrustLevel_1:
		ok := false
		// [~server.design.sequences/tuc.SequencesTrustLevelForPLog~impl]
		if ok, err = e.app.config.storage.InsertIfNotExists(pKey, cCols, evData, 0); err == nil {
			if !ok {
				return nil, ErrSequencesViolation
			}
		}
	default:
		// notest
		panic("unexpected SequencesTrustLevel " + strconv.Itoa(int(e.app.seqTrustLevel)))
	}
	dbEvent.isStored = true
	if err == nil {
		event = dbEvent
		e.plogCache.Put(p, o, event)
	}

	return event, err
}

func getEventBytes(ev istructs.IPLogEvent) (pKey, cCols, data []byte) {
	pKey, cCols = wlogKey(ev.Workspace(), ev.WLogOffset())
	data = ev.(*eventType).storeToBytes()
	return pKey, cCols, data
}

// istructs.IEvents.PutWlog
func (e *appEventsType) PutWlog(ev istructs.IPLogEvent) (err error) {
	pKey, cCols, evData := getEventBytes(ev)
	switch {
	case ev.QName() == istructs.QNameForCorruptedData, e.app.seqTrustLevel == isequencer.SequencesTrustLevel_2:
		err = e.app.config.storage.Put(pKey, cCols, evData)
	case e.app.seqTrustLevel == isequencer.SequencesTrustLevel_0, e.app.seqTrustLevel == isequencer.SequencesTrustLevel_1:
		ok := false
		// [~server.design.sequences/tuc.SequencesTrustLevelForWLog~impl]
		if ok, err = e.app.config.storage.InsertIfNotExists(pKey, cCols, evData, 0); err == nil {
			if !ok {
				return ErrSequencesViolation
			}
		}
	default:
		// notest
		panic("unexpected SequencesTrustLevel " + strconv.Itoa(int(e.app.seqTrustLevel)))
	}
	return err
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
		return ErrMaxGetBatchSizeExceeds(len(ids))
	}
	batches := make([]*istorage.GetBatchItem, len(ids))
	plan := make(map[string][]istorage.GetBatchItem)
	for i := range ids {
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
	for i := range batches {
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
	id    istructs.RecordID
	data  []byte
	isNew bool
}

func (recs *appRecordsType) putRecordsBatch(workspace istructs.WSID, records []recordBatchItemType, isReapply bool) (err error) {
	batch := make([]istorage.BatchItem, len(records))
	switch {
	case isReapply, recs.app.seqTrustLevel == isequencer.SequencesTrustLevel_1, recs.app.seqTrustLevel == isequencer.SequencesTrustLevel_2:
		for i, r := range records {
			batch[i].PKey, batch[i].CCols = recordKey(workspace, r.id)
			batch[i].Value = r.data
		}
		return recs.app.config.storage.PutBatch(batch)
	case recs.app.seqTrustLevel == isequencer.SequencesTrustLevel_0:
		for _, r := range records {
			pKey, cCols := recordKey(workspace, r.id)
			// [~tuc.SequencesTrustLevelForRecords~]
			if r.isNew {
				ok, err := recs.app.config.storage.InsertIfNotExists(pKey, cCols, r.data, 0)
				if err != nil {
					// notest
					return err
				}
				if !ok {
					return ErrSequencesViolation
				}
			} else {
				if err := recs.app.config.storage.Put(pKey, cCols, r.data); err != nil {
					// notest
					return err
				}
			}
		}
	default:
		// notest
		panic("unexpected SequencesTrustLevel " + strconv.Itoa(int(recs.app.seqTrustLevel)))
	}
	return nil
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
				return enrichError(err, "error checking singleton «%v» record «%d» existence", rec.QName(), id)
			}
			if exists {
				return ErrSingletonViolation(rec)
			}
		}
	}

	for _, rec := range ev.cud.updates {
		old := newRecord(recs.app.config)
		exists, err := load(rec.originRec.ID(), old)
		if err != nil {
			return enrichError(err, "error load updated «%v» record «%d»", rec.originRec.QName(), rec.originRec.ID())
		}
		if !exists {
			return ErrIDNotFound("updated «%v» record «%d»", rec.originRec.QName(), rec.originRec.ID())
		}

		// check exists record has correct QName
		if rec.originRec.QName() != old.QName() {
			return ErrWrongType("updated «%v» record «%d» has unexpected QName value «%v»", rec.originRec.QName(), rec.originRec.ID(), old.QName())
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
	return recs.apply2(event, cb, false)
}

func (recs *appRecordsType) apply2(event istructs.IPLogEvent, cb func(rec istructs.IRecord), isReapply bool) (err error) {
	ev := event.(*eventType)

	if !ev.Error().ValidEvent() {
		panic(ErrorEventNotValid("can not apply not valid event: %s:", ev.Error().ErrStr()))
	}

	records := make([]*recordType, 0)
	batch := make([]recordBatchItemType, 0)

	store := func(rec *recordType) error {
		data := rec.storeToBytes()
		records = append(records, rec)
		batch = append(batch, recordBatchItemType{rec.ID(), data, rec.isNew})

		return nil
	}

	load := func(rec *recordType) error {
		data := make([]byte, 0)
		exists, err := recs.getRecord(ev.ws, rec.ID(), &data)
		if err != nil {
			return err
		}
		if !exists {
			return ErrIDNotFound("record «%d»", rec.ID())
		}
		return rec.loadFromBytes(data)
	}

	if err = ev.cud.applyRecs(load, store); err == nil {
		if len(records) > 0 {
			if err = recs.putRecordsBatch(ev.ws, batch, isReapply); err == nil {
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
func (recs *appRecordsType) Get(workspace istructs.WSID, _ bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	data := make([]byte, 0)
	if ok, err := recs.getRecord(workspace, id, &data); !ok {
		if err != nil {
			// storage error while getRecord
			return NewNullRecord(id), enrichError(err, "ws: %d, id: %d", workspace, id)
		}
		return NewNullRecord(id), nil // just not found, no error
	}
	rec := newRecord(recs.app.config)
	if err := rec.loadFromBytes(data); err != nil {
		return NewNullRecord(id), enrichError(err, "ws: %d, id: %d", workspace, id)
	}
	return rec, nil
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
		return ErrWrongType("%v is not storable record type", rec.typeDef())
	}

	id := rec.ID()
	if id == istructs.NullRecordID {
		return ErrFieldMissed(rec, appdef.SystemField_ID)
	}
	if id.IsRaw() {
		return ErrUnexpectedRawRecordID(rec, appdef.SystemField_ID, id)
	}

	return recs.putRecord(ws, id, rec.storeToBytes())
}
