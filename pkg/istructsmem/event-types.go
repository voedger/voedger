/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

type (
	existsRecordType    func(id istructs.RecordID) (bool, error)
	loadRecordFuncType  func(rec *recordType) error
	storeRecordFuncType func(rec *recordType) error
)

// eventType implements event structure
//   - interfaces:
//     — istructs.IRawEventBuilder
//     — istructs.IAbstractEvent
//     — istructs.IRawEvent
type eventType struct {
	appCfg    *AppConfigType
	rawBytes  []byte
	partition istructs.PartitionID
	pLogOffs  istructs.Offset
	ws        istructs.WSID
	wLogOffs  istructs.Offset
	name      istructs.QName
	regTime   istructs.UnixMilli
	sync      bool
	device    istructs.ConnectedDeviceID
	syncTime  istructs.UnixMilli
	argObject elementType
	argUnlObj elementType
	cud       cudType
}

// newRawEvent creates new empty raw event
func newRawEvent(appCfg *AppConfigType) eventType {
	event := eventType{
		appCfg:    appCfg,
		argObject: newObject(appCfg, istructs.NullQName),
		argUnlObj: newObject(appCfg, istructs.NullQName),
		cud:       newCUD(appCfg),
	}
	return event
}

// newEventParams creates new empty raw event with specified params
func newEventParams(appCfg *AppConfigType, params istructs.GenericRawEventBuilderParams) eventType {
	ev := newRawEvent(appCfg)
	ev.rawBytes = make([]byte, len(params.EventBytes))
	copy(ev.rawBytes, params.EventBytes)
	ev.partition = params.HandlingPartition
	ev.pLogOffs = params.PLogOffset
	ev.ws = params.Workspace
	ev.wLogOffs = params.WLogOffset
	ev.setName(params.QName)
	ev.regTime = params.RegisteredAt
	return ev
}

// newEvent creates new raw event
func newEvent(appCfg *AppConfigType, params istructs.NewRawEventBuilderParams) eventType {
	return newEventParams(appCfg, params.GenericRawEventBuilderParams)
}

// newSyncEvent creates new synced raw event
func newSyncEvent(appCfg *AppConfigType, params istructs.SyncRawEventBuilderParams) eventType {
	ev := newEventParams(appCfg, params.GenericRawEventBuilderParams)
	ev.sync = true
	ev.device = params.Device
	ev.syncTime = params.SyncedAt
	return ev
}

// argumentNames returns argnument and unlogged argument QNames
func (ev *eventType) argumentNames() (arg, argUnl istructs.QName, err error) {
	arg = istructs.NullQName
	argUnl = istructs.NullQName

	if ev.name == istructs.QNameCommandCUD {
		return arg, argUnl, nil // #17664 — «sys.CUD» command has no arguments objects, only CUDs
	}

	cmd := ev.appCfg.Resources.CommandFunction(ev.name)
	if cmd != nil {
		arg = cmd.ParamsSchema()
		argUnl = cmd.UnloggedParamsSchema()
	} else {
		// #!16208: Must be possible to use SchemaKind_ODoc as Event.QName
		if schema := ev.appCfg.Schemas.Schema(ev.name); schema.Kind() != istructs.SchemaKind_ODoc {
			return arg, argUnl, fmt.Errorf("command function «%v» not found: %w", ev.name, ErrNameNotFound)
		}
		arg = ev.name
	}

	return arg, argUnl, nil
}

// build build all event arguments and CUDs
func (ev *eventType) build() (err error) {
	if ev.name == istructs.NullQName {
		return validateErrorf(ECode_EmptySchemaName, "empty event command name: %w", ErrNameMissed)
	}

	if _, err = ev.appCfg.qNames.qNameToID(ev.name); err != nil {
		return validateErrorf(ECode_InvalidSchemaName, "unknown event command name «%v»: %w", ev.name, err)
	}

	if err = ev.argObject.build(); err != nil {
		return err
	}
	if err = ev.argUnlObj.build(); err != nil {
		return err
	}
	if err = ev.cud.build(); err != nil {
		return err
	}

	return nil
}

// copyFrom copies event from specified source
func (ev *eventType) copyFrom(src *eventType) {
	ev.appCfg = src.appCfg

	ev.rawBytes = make([]byte, len(src.rawBytes))
	copy(ev.rawBytes, src.rawBytes)
	ev.partition = src.HandlingPartition()
	ev.pLogOffs = src.PLogOffset()
	ev.ws = src.Workspace()
	ev.wLogOffs = src.WLogOffset()
	ev.setName(src.QName())
	ev.regTime = src.RegisteredAt()
	ev.sync = src.Synced()
	ev.device = src.DeviceID()
	ev.syncTime = src.SyncedAt()

	ev.argObject.copyFrom(&src.argObject)
	ev.argUnlObj.copyFrom(&src.argUnlObj)
	ev.cud.copyFrom(&src.cud)
}

// regenerateIDs regenerates all raw IDs in event arguments and CUDs using specified generator
func (ev *eventType) regenerateIDs(generator istructs.IDGenerator) (err error) {
	if (ev.argObject.QName() != istructs.NullQName) && ev.argObject.isDocument() {
		if err := ev.argObject.regenerateIDs(generator); err != nil {
			return err
		}
	}

	if err := ev.cud.regenerateIDs(generator); err != nil {
		return err
	}
	return nil
}

// setName sets specified command name for event. Command name may be ODOC name, see #!16208
func (ev *eventType) setName(n istructs.QName) {
	ev.name = n
	if ev.appCfg != nil {
		if arg, argUnl, err := ev.argumentNames(); err == nil {
			ev.argObject.setQName(arg)
			ev.argUnlObj.setQName(argUnl)
		}
	}
}

// istructs.IRawEventBuilder.ArgumentObjectBuilder() IObjectBuilder
func (ev *eventType) ArgumentObjectBuilder() istructs.IObjectBuilder {
	return &ev.argObject
}

// istructs.IRawEventBuilder.UnloggedArgumentObjectBuilder() IObjectBuilder
func (ev *eventType) ArgumentUnloggedObjectBuilder() istructs.IObjectBuilder {
	return &ev.argUnlObj
}

// istructs.IRawEventBuilder.CUDBuilder
func (ev *eventType) CUDBuilder() istructs.ICUD {
	return &ev.cud
}

// istructs.IRawEventBuilder.BuildRawEvent
func (ev *eventType) BuildRawEvent() (raw istructs.IRawEvent, err error) {
	if err = ev.build(); err != nil {
		return ev, err
	}

	if err = ev.appCfg.Schemas.validEvent(ev); err != nil {
		return ev, err
	}

	if err = ev.appCfg.app.records.validEvent(ev); err != nil {
		return ev, err
	}

	return ev, nil
}

// istructs.IAbstractEvent.QName. Be careful — this method is overridden by dbEventType
func (ev *eventType) QName() istructs.QName {
	return ev.name
}

// istructs.IAbstractEvent.ArgumentObject
func (ev *eventType) ArgumentObject() istructs.IObject {
	return &ev.argObject
}

// istructs.IAbstractEvent.CUDs
func (ev *eventType) CUDs(cb func(rec istructs.ICUDRow) error) (err error) {
	return ev.cud.enumRecs(cb)
}

// istructs.IAbstractEvent.RegiseredAt
func (ev *eventType) RegisteredAt() istructs.UnixMilli {
	return ev.regTime
}

// istructs.IAbstractEvent.Synced
func (ev *eventType) Synced() bool {
	return ev.sync
}

// istructs.IAbstractEvent.DeviceID
func (ev *eventType) DeviceID() istructs.ConnectedDeviceID {
	if !ev.sync {
		return 0
	}
	return ev.device
}

// istructs.IAbstractEvent.SyncedAt
func (ev *eventType) SyncedAt() istructs.UnixMilli {
	if !ev.sync {
		return 0
	}
	return ev.syncTime
}

// istructs.IRawEvent.ArgumentUnloggedObject //
func (ev *eventType) ArgumentUnloggedObject() istructs.IObject {
	return &ev.argUnlObj
}

// istructs.IRawEvent.HandlingPartition
func (ev *eventType) HandlingPartition() istructs.PartitionID {
	return ev.partition
}

// istructs.IRawEvent.PLogOffset
func (ev *eventType) PLogOffset() istructs.Offset {
	return ev.pLogOffs
}

// istructs.IRawEvent.Workspace
func (ev *eventType) Workspace() istructs.WSID {
	return ev.ws
}

// istructs.IRawEvent.WLogOffset
func (ev *eventType) WLogOffset() istructs.Offset {
	return ev.wLogOffs
}

// dbEventType Implements storable into DB event
//   - interfaces:
//     — istructs.IDbEvent,
//     — istructs.IPLogEvent,
//     — istructs.IWLogEvent
type dbEventType struct {
	eventType
	buildErr eventErrorType
}

// newDbEvent creates an empty DB event
func newDbEvent(appCfg *AppConfigType) (ev dbEventType) {
	event := dbEventType{
		eventType: newRawEvent(appCfg),
	}
	event.buildErr = newEventError()
	return event
}

// applyCommandRecs store all event CUDs into storage records using specified cb functions
func (ev *dbEventType) applyCommandRecs(exists existsRecordType, load loadRecordFuncType, store storeRecordFuncType) error {
	return ev.cud.applyRecs(exists, load, store)
}

// copyFrom copies members from source
func (ev *dbEventType) copyFrom(src *dbEventType) {
	ev.eventType.copyFrom(&src.eventType)
	ev.buildErr.copyFrom(&src.buildErr)
}

// loadFromBytes loads event from bytes and returns error if occurced
func (ev *dbEventType) loadFromBytes(in []byte) (err error) {
	buf := bytes.NewBuffer(in)
	var codec byte
	if err = binary.Read(buf, binary.BigEndian, &codec); err != nil {
		return fmt.Errorf("error read codec version: %w", err)
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1:
		if err := loadEvent(ev, codec, buf); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown codec version «%d»: %w", codec, ErrUnknownCodec)
	}

	return nil
}

// qNameID retrieves ID for event command name
func (ev *dbEventType) qNameID() QNameID {
	if ev.valid() {
		if id, err := ev.appCfg.qNames.qNameToID(ev.QName()); err == nil {
			return id
		}
	}
	return QNameIDForError
}

// setBuildError sets specified error as build event error
func (ev *dbEventType) setBuildError(err error) {
	ev.buildErr.setError(ev, err)
}

// storeToBytes stores event into bytes slice and returns error if occurced
func (ev *dbEventType) storeToBytes() (out []byte, err error) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, codec_LastVersion)

	if err = storeEvent(ev, buf); err == nil {
		out = buf.Bytes()
	}

	return out, err
}

func (ev *dbEventType) valid() bool {
	return ev.buildErr.validEvent
}

// istructs.IDbEvent.Error
func (ev *dbEventType) Error() istructs.IEventError {
	return &ev.buildErr
}

// istructs.IDbEvent.QName — overrides IAbstractEvent.QName()
func (ev *dbEventType) QName() istructs.QName {
	qName := istructs.QNameForError
	if ev.valid() {
		qName = ev.name
	}
	return qName
}

// istructs.IPLogEvent.Release and IWLogEvent.Release
func (ev *dbEventType) Release() {}

// cudType implements event cud member
//   - methods:
//     — regenerateIDs: regenerates all raw IDs by specified generator
//     — validRawIDs: validates raw IDs and refers to raw IDs
//   - interfaces:
//     — istructs.ICUD
type cudType struct {
	appCfg  *AppConfigType
	creates []*recordType
	updates map[istructs.RecordID]*updateRecType
}

func newCUD(appCfg *AppConfigType) cudType {
	return cudType{
		appCfg:  appCfg,
		creates: make([]*recordType, 0),
		updates: make(map[istructs.RecordID]*updateRecType),
	}
}

// applyRecs call store callback func for each record
func (cud *cudType) applyRecs(exists existsRecordType, load loadRecordFuncType, store storeRecordFuncType) (err error) {

	for _, rec := range cud.creates {
		if rec.schema.singleton.enabled {
			isExists, err := exists(rec.schema.singleton.id)
			if err != nil {
				return err
			}
			if isExists {
				return fmt.Errorf("can not create singleton, CDOC «%v» record «%d» already exists: %w", rec.QName(), rec.schema.singleton.id, ErrRecordIDUniqueViolation)
			}
		}
		if err = store(rec); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if rec.originRec.empty() {
			if err = load(&rec.originRec); err != nil {
				return err
			}
			if err = rec.build(); err != nil {
				return err
			}
		}
		if err = store(&rec.result); err != nil {
			return err
		}
	}

	return nil // all is ok
}

// build builds creates and updates and returns error if occurs
func (cud *cudType) build() (err error) {
	for _, rec := range cud.creates {
		if err = rec.build(); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if err = rec.build(); err != nil {
			return err
		}
	}
	return nil
}

// copyFrom copies members from source
func (cud *cudType) copyFrom(src *cudType) {
	cud.creates = make([]*recordType, len(src.creates))
	for i, srcRec := range src.creates {
		rec := newRecord(cud.appCfg)
		cud.creates[i] = &rec
		rec.copyFrom(srcRec)
	}

	cud.updates = make(map[istructs.RecordID]*updateRecType)
	for id, srcRec := range src.updates {
		rec := newUpdateRec(cud.appCfg, &srcRec.originRec)
		cud.updates[id] = &rec
		rec.copyFrom(srcRec)
	}
}

// empty return is all members is empty
func (cud *cudType) empty() bool {
	return (len(cud.creates) == 0) && (len(cud.updates) == 0)
}

// enumRecs: enumerates changes as IRecords
func (cud *cudType) enumRecs(cb func(rec istructs.ICUDRow) error) (err error) {
	for _, rec := range cud.creates {
		if err = cb(rec); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if err = cb(&rec.changes); err != nil { // changed fields only
			return err
		}
	}

	return nil
}

// newIDsPlanType is type for ID regeneration plan. Key is raw ID, value is storage ID
type newIDsPlanType map[istructs.RecordID]istructs.RecordID

// regenerateIDsPlan creates new ID regeneration plan
func (cud *cudType) regenerateIDsPlan(generator istructs.IDGenerator) (newIDs newIDsPlanType, err error) {
	plan := make(newIDsPlanType)
	for _, rec := range cud.creates {
		id := rec.ID()
		if !id.IsRaw() {
			continue // storage IDs is allowed for sync events…
		}

		var storeID istructs.RecordID

		if rec.schema.singleton.enabled {
			storeID = rec.schema.singleton.id
		} else {
			if storeID, err = generator(id, rec.schema); err != nil {
				return nil, err
			}
		}

		rec.setID(storeID)
		plan[id] = storeID
	}
	return plan, nil
}

// regenerateIDsInRecord regenerates ID in single record using specified plan
func regenerateIDsInRecord(rec *recordType, newIDs newIDsPlanType) (err error) {
	changes := false

	rec.RecordIDs(false, func(name string, value istructs.RecordID) {
		if !value.IsRaw() {
			return
		}
		if id, ok := newIDs[value]; ok {
			rec.PutRecordID(name, id)
			changes = true
		}
	})
	if changes {
		// record must be rebuilded to apply changes to dynobuffer
		err = rec.build()
	}
	return err
}

// regenerateIDsInUpdateRecord regenerates ID in single CUD update record changes using specified plan
func regenerateIDsInUpdateRecord(rec *updateRecType, newIDs newIDsPlanType) (err error) {
	changes := false

	rec.changes.RecordIDs(false, func(name string, value istructs.RecordID) {
		if !value.IsRaw() {
			return
		}
		if id, ok := newIDs[value]; ok {
			rec.changes.PutRecordID(name, id)
			changes = true
		}
	})

	if changes {
		// record (changes and result) must be rebuilded to apply changes to dynobuffer
		err = rec.build()
	}
	return err
}

// regenerateIDs regerates all raw IDs to storage IDs
func (cud *cudType) regenerateIDs(generator istructs.IDGenerator) error {

	newIDs, err := cud.regenerateIDsPlan(generator)
	if err != nil {
		return err
	}

	for _, rec := range cud.creates {
		if err = regenerateIDsInRecord(rec, newIDs); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if err = regenerateIDsInUpdateRecord(rec, newIDs); err != nil {
			return err
		}
	}

	return nil
}

// istructs.ICUD.Create
func (cud *cudType) Create(qName istructs.QName) istructs.IRowWriter {
	r := newRecord(cud.appCfg)
	r.isNew = true
	r.setQName(qName)
	rec := &r

	cud.creates = append(cud.creates, rec)

	return rec
}

// istructs.ICUD.Update
func (cud *cudType) Update(record istructs.IRecord) istructs.IRowWriter {
	id := record.ID()
	rec, ok := cud.updates[id]
	if !ok {
		r := newUpdateRec(cud.appCfg, record)
		rec = &r
		cud.updates[id] = rec
	}

	return &rec.changes
}

// updateRecType is plan to update record
type updateRecType struct {
	appCfg    *AppConfigType
	originRec recordType
	changes   recordType
	result    recordType
}

func newUpdateRec(appCfg *AppConfigType, rec istructs.IRecord) updateRecType {
	upd := updateRecType{
		appCfg:    appCfg,
		originRec: newRecord(appCfg),
		changes:   newRecord(appCfg),
		result:    newRecord(appCfg),
	}
	upd.originRec.copyFrom(rec.(*recordType))

	upd.changes.setQName(rec.QName())
	upd.changes.setID(rec.ID())

	upd.changes.setParent(rec.Parent())
	upd.changes.setContainer(rec.Container())
	if r, ok := rec.(*recordType); ok {
		upd.changes.setActive(r.IsActive())
	}

	upd.result.copyFrom(&upd.originRec)

	return upd
}

// build builds record changes and applies them to result record. If no errors then builds result record
func (upd *updateRecType) build() (err error) {

	upd.result.copyFrom(&upd.originRec)

	if upd.changes.QName() == istructs.NullQName {
		return nil
	}

	if err = upd.changes.build(); err != nil {
		return err
	}

	if upd.originRec.ID() != upd.changes.ID() {
		return fmt.Errorf("record «%v» ID «%d» can not to be updated: %w", upd.originRec.QName(), upd.originRec.ID(), ErrUnableToUpdateSystemField)
	}
	if (upd.changes.Parent() != istructs.NullRecordID) && (upd.changes.Parent() != upd.originRec.Parent()) {
		return fmt.Errorf("record «%v» parent ID «%d» can not to be updated: %w", upd.originRec.QName(), upd.originRec.Parent(), ErrUnableToUpdateSystemField)
	}
	if (upd.changes.Container() != "") && (upd.changes.Container() != upd.originRec.Container()) {
		return fmt.Errorf("record «%v» container «%s» can not to be updated: %w", upd.originRec.QName(), upd.originRec.Container(), ErrUnableToUpdateSystemField)
	}

	if upd.changes.IsActive() != upd.originRec.IsActive() {
		upd.result.setActive(upd.changes.IsActive())
	}

	userChanges := false
	upd.changes.dyB.IterateFields(nil, func(name string, newData interface{}) bool {
		upd.result.dyB.Set(name, newData)
		userChanges = true
		return true
	})

	if userChanges {
		err = upd.result.build()
	}

	return err
}

// copyFrom copies members from source
func (upd *updateRecType) copyFrom(src *updateRecType) {
	upd.changes.copyFrom(&src.changes)
	upd.originRec.copyFrom(&src.originRec)
	upd.result.copyFrom(&src.result)
}

// elementType implements object and element (as part of object) structure
//   - interfaces:
//     — istructs.IObjectBuilder
//     — istructs.IElementBuilder
//     — istructs.IObject,
//     — istructs.IElement
type elementType struct {
	recordType
	parent *elementType
	childs []*elementType
}

func newObject(appCfg *AppConfigType, qn istructs.QName) elementType {
	obj := elementType{
		recordType: newRecord(appCfg),
		childs:     make([]*elementType, 0),
	}
	obj.setQName(qn)
	return obj
}

func newElement(parent *elementType) elementType {
	el := elementType{
		recordType: newRecord(parent.appCfg),
		parent:     parent,
		childs:     make([]*elementType, 0),
	}
	return el
}

// build builds element record and all childs recursive
func (el *elementType) build() (err error) {
	return el.forEach(func(e *elementType) error {
		return e.rowType.build()
	})
}

// clear clears element record and all childs recursive
func (el *elementType) clear() {
	el.recordType.clear()
	el.childs = make([]*elementType, 0)
}

// copyFrom copies element record row and clone all childs hierarchy recursive
func (el *elementType) copyFrom(src *elementType) {
	el.clear()
	el.recordType.copyFrom(&src.recordType)
	for _, srcC := range src.childs {
		c := newElement(el)
		c.copyFrom(srcC)
		el.childs = append(el.childs, &c)
	}
}

// forEach applies cb function to element and all it childs recursive
func (el *elementType) forEach(cb func(e *elementType) error) (err error) {
	if err = cb(el); err == nil {
		for _, e := range el.childs {
			if err = e.forEach(cb); err != nil {
				break
			}
		}
	}
	return err
}

// isDocument returns is document schema assigned to element record
func (el *elementType) isDocument() bool {
	kind := el.schema.Kind()
	return (kind == istructs.SchemaKind_GDoc) ||
		(kind == istructs.SchemaKind_CDoc) ||
		(kind == istructs.SchemaKind_ODoc) ||
		(kind == istructs.SchemaKind_WDoc)
}

// maskValues masks element record row values and all elements chils recursive
func (el *elementType) maskValues() {
	el.rowType.maskValues()

	for _, e := range el.childs {
		e.maskValues()
	}
}

// regenerateIDs regenerates element record IDs and all elements childs recursive.
// If some child record ID reference (e.c. «sys.Parent» fields) refers to regenerated parent ID fields, this replaced too.
func (el *elementType) regenerateIDs(generator istructs.IDGenerator) (err error) {
	newIDs := make(newIDsPlanType)

	err = el.forEach(
		func(e *elementType) error {
			if id := e.ID(); id.IsRaw() {
				storeID, err := generator(id, e.schema)
				if err != nil {
					return err
				}
				e.setID(storeID)
				newIDs[id] = storeID
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = el.forEach(
		func(e *elementType) (err error) {
			if id := e.Parent(); id.IsRaw() {
				e.setParent(newIDs[id])
			}

			changes := false
			e.RecordIDs(false, func(name string, id istructs.RecordID) {
				if id.IsRaw() {
					e.PutRecordID(name, newIDs[id])
					changes = true
				}
			})
			if changes {
				// element must be rebuilded to apply changes in dynobuffer
				err = e.build()
			}
			return err
		})

	return err
}

// istructs.IElementBuilder.ElementBuilder
func (el *elementType) ElementBuilder(containerName string) istructs.IElementBuilder {
	c := newElement(el)
	el.childs = append(el.childs, &c)
	if el.QName() != istructs.NullQName {
		c.setQName(el.schema.containerQName(containerName))

		if c.QName() != istructs.NullQName {
			if el.ID() != istructs.NullRecordID {
				c.setParent(el.ID())
			}
			c.setContainer(containerName)
		}
	}
	return &c
}

// istructs.IElement.Elements
func (el *elementType) Elements(container string, cb func(nestedPart istructs.IElement)) {
	for _, c := range el.childs {
		if c.Container() == container {
			cb(c)
		}
	}
}

// istructs.IElement.Containers
func (el *elementType) Containers(cb func(container string)) {
	dups := make(map[string]bool, len(el.childs))
	for _, c := range el.childs {
		name := c.Container()
		if dups[name] {
			continue
		}
		cb(name)
		dups[name] = true
	}
}

// istructs.IObjectBuilder.Build()
func (el *elementType) Build() (doc istructs.IObject, err error) {
	if err = el.build(); err != nil {
		return nil, err
	}
	if err = el.appCfg.Schemas.validObject(el); err != nil {
		return nil, err
	}

	return el, nil
}

// istructs.IElement.QName()
func (el *elementType) QName() istructs.QName {
	return el.recordType.QName()
}

// istructs.IObject.AsRecord()
func (el *elementType) AsRecord() istructs.IRecord {
	return el
}

// eventErrorType implemnts IEventError
//   - interfaces
//     — istructs.IEventError
type eventErrorType struct {
	validEvent bool
	errStr     string
	qName      istructs.QName
	bytes      []byte
}

func newEventError() eventErrorType {
	return eventErrorType{
		validEvent: true,
		qName:      istructs.NullQName,
	}
}

// copyFrom copies members from source
func (e *eventErrorType) copyFrom(src *eventErrorType) {
	e.validEvent = src.validEvent
	e.errStr = src.errStr
	e.qName = src.qName
	e.bytes = src.bytes
}

// setError sets event build error
func (e *eventErrorType) setError(event *dbEventType, err error) {
	if err == nil {
		e.validEvent = true
		e.errStr = ""
		e.qName = istructs.NullQName
		e.bytes = nil
	} else {
		e.validEvent = false
		e.errStr = err.Error()
		e.qName = event.name
		e.bytes = make([]byte, len(event.rawBytes))
		copy(e.bytes, event.rawBytes)
	}
}

// istructs.IEventError.ErrStr
func (e *eventErrorType) ErrStr() string {
	return e.errStr
}

// istructs.IEventError.QNameFromParams
func (e *eventErrorType) QNameFromParams() istructs.QName {
	return e.qName
}

// istructs.IEventError.ValidEvent
func (e *eventErrorType) ValidEvent() bool {
	return e.validEvent
}

// istructs.IEventError.OriginalEventBytes
func (e *eventErrorType) OriginalEventBytes() []byte {
	return e.bytes
}
