/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	bytespool "github.com/valyala/bytebufferpool"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/objcache"
)

type recordFunc func(rec *recordType) error

// Implements event structure
//
//	# Implemented interfaces:
//	   — istructs.IRawEventBuilder
//	   — istructs.IAbstractEvent
//	   — istructs.IRawEvent
//
//	   — istructs.IDbEvent,
//	   — istructs.IPLogEvent,
//	   — istructs.IWLogEvent
type eventType struct {
	appCfg    *AppConfigType
	rawBytes  []byte
	partition istructs.PartitionID
	pLogOffs  istructs.Offset
	ws        istructs.WSID
	wLogOffs  istructs.Offset
	name      appdef.QName
	regTime   istructs.UnixMilli
	sync      bool
	device    istructs.ConnectedDeviceID
	syncTime  istructs.UnixMilli
	argObject objectType
	argUnlObj objectType
	cud       cudType

	// db event members
	buildErr eventErrorType

	buffer *bytespool.ByteBuffer

	// cache supports
	objcache.RefCounter
}

// Returns new empty event
func newEvent(appCfg *AppConfigType) *eventType {
	event := &eventType{
		appCfg:    appCfg,
		argObject: makeObject(appCfg, appdef.NullQName, nil),
		argUnlObj: makeObject(appCfg, appdef.NullQName, nil),
		cud:       makeCUD(appCfg),
		buildErr:  makeEventError(),
	}
	event.RefCounter.Value = event
	return event
}

// Returns new empty raw event with specified params
func newRawEventBuilder(appCfg *AppConfigType, params istructs.GenericRawEventBuilderParams) *eventType {
	ev := newEvent(appCfg)

	ev.rawBytes = utils.CopyBytes(params.EventBytes)

	ev.partition = params.HandlingPartition
	ev.pLogOffs = params.PLogOffset
	ev.ws = params.Workspace
	ev.wLogOffs = params.WLogOffset
	ev.setName(params.QName)
	ev.regTime = params.RegisteredAt

	if ev.name == istructs.QNameForCorruptedData {
		ev.buildErr.setError(ev, ErrCorruptedData)
	}

	return ev
}

// Returns new raw event builder
func newEventBuilder(appCfg *AppConfigType, params istructs.NewRawEventBuilderParams) *eventType {
	return newRawEventBuilder(appCfg, params.GenericRawEventBuilderParams)
}

// Returns new synced raw event builder
func newSyncEventBuilder(appCfg *AppConfigType, params istructs.SyncRawEventBuilderParams) *eventType {
	ev := newRawEventBuilder(appCfg, params.GenericRawEventBuilderParams)
	ev.sync = true
	ev.device = params.Device
	ev.syncTime = params.SyncedAt
	return ev
}

// argumentNames returns argument and un-logged argument QNames
func (ev *eventType) argumentNames() (arg, argUnl appdef.QName, err error) {
	arg = appdef.NullQName
	argUnl = appdef.NullQName

	if ev.name == istructs.QNameCommandCUD {
		return arg, argUnl, nil // #17664 — «sys.CUD» command has no arguments objects, only CUDs
	}

	if ev.name == istructs.QNameForCorruptedData {
		return arg, argUnl, nil // #1811 — «sys.Corrupted» command has no arguments objects
	}

	cmd := appdef.Command(ev.appCfg.AppDef.Type, ev.name)
	if cmd != nil {
		if cmd.Param() != nil {
			arg = cmd.Param().QName()
		}
		if cmd.UnloggedParam() != nil {
			argUnl = cmd.UnloggedParam().QName()
		}
	} else {
		// #!16208: Should be possible to use TypeKind_ODoc as Event.QName
		if d := appdef.ODoc(ev.appCfg.AppDef.Type, ev.name); d == nil {
			// command function «test.object» not found
			return arg, argUnl, ErrNameNotFound("command function «%v»", ev.name)
		}
		arg = ev.name
	}

	return arg, argUnl, nil
}

// build build all event arguments and CUDs
func (ev *eventType) build() (err error) {
	if ev.name == appdef.NullQName {
		return validateError(ECode_EmptyTypeName, ErrNameMissed("empty event command name"))
	}

	if _, err = ev.appCfg.qNames.ID(ev.name); err != nil {
		return validateErrorf(ECode_InvalidTypeName, "unknown event command name «%v»: %w", ev.name, err)
	}

	err = errors.Join(
		ev.argObject.build(),
		ev.argUnlObj.build(),
		ev.cud.build(),
	)

	return err
}

// Loads event from bytes and returns error if occurs
func (ev *eventType) loadFromBytes(in []byte) (err error) {
	buf := bytes.NewBuffer(in)
	var codec byte
	if codec, err = utils.ReadByte(buf); err != nil {
		return fmt.Errorf("error read codec version: %w", err)
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1, codec_RDB_2:
		if err := loadEvent(ev, codec, buf); err != nil {
			return err
		}
	default:
		return ErrUnknownCodec(codec)
	}

	return nil
}

// Retrieves ID for event command name
func (ev *eventType) qNameID() (id qnames.QNameID) {
	if id, err := ev.appCfg.qNames.ID(ev.QName()); err == nil {
		return id
	}
	// no test
	return qnames.QNameIDForError
}

// Regenerates all raw IDs in event arguments and CUDs using specified generator
func (ev *eventType) regenerateIDs(generator istructs.IIDGenerator) (err error) {
	if (ev.argObject.QName() != appdef.NullQName) && ev.argObject.isDocument() {
		if err := ev.argObject.regenerateIDs(generator); err != nil {
			return err
		}
	}

	if err := ev.cud.regenerateIDs(generator); err != nil {
		return err
	}
	return nil
}

// Sets specified error as build event error
func (ev *eventType) setBuildError(err error) {
	ev.buildErr.setError(ev, err)
}

// Sets specified command name for event. Command name may be ODoc name, see #!16208
func (ev *eventType) setName(n appdef.QName) {
	ev.name = n
	if ev.appCfg != nil {
		if arg, argUnl, err := ev.argumentNames(); err == nil {
			ev.argObject.setQName(arg)
			ev.argUnlObj.setQName(argUnl)
		}
	}
}

// Stores event into bytes slice
//
// # Panics:
//
//   - Must be called *after* event validation. Overwise function may panic!
func (ev *eventType) storeToBytes() []byte {
	if ev.buffer == nil {
		ev.buffer = bytespool.Get()
		buf := bytes.NewBuffer(ev.buffer.B)
		utils.WriteByte(buf, codec_LastVersion)
		storeEvent(ev, buf)
		ev.buffer.B = buf.Bytes()
	}
	return ev.buffer.B
}

// Returns is event valid.
//
// sys.Corrupted event is always invalid
func (ev *eventType) valid() bool {
	if ev.name == istructs.QNameForCorruptedData {
		return false
	}
	return ev.buildErr.validEvent
}

// istructs.IRawEventBuilder.ArgumentObjectBuilder() IObjectBuilder
func (ev *eventType) ArgumentObjectBuilder() istructs.IObjectBuilder {
	return &ev.argObject
}

// istructs.IRawEventBuilder.UnloggedArgumentObjectBuilder() IObjectBuilder
func (ev *eventType) ArgumentUnloggedObjectBuilder() istructs.IObjectBuilder {
	return &ev.argUnlObj
}

// istructs.IDBEvent.Bytes
func (ev *eventType) Bytes() []byte {
	if ev.buffer != nil {
		return ev.buffer.B
	}
	return nil
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

	if err = validateEvent(ev); err != nil {
		return ev, err
	}

	if err = ev.appCfg.app.records.validEvent(ev); err != nil {
		return ev, err
	}

	return ev, nil
}

// istructs.IAbstractEvent.ArgumentObject
func (ev *eventType) ArgumentObject() istructs.IObject {
	return &ev.argObject
}

// istructs.IAbstractEvent.CUDs
func (ev *eventType) CUDs(cb func(rec istructs.ICUDRow) bool) {
	ev.cud.enumRecs(cb)
}

// istructs.IDbEvent.Error
func (ev *eventType) Error() istructs.IEventError {
	return &ev.buildErr
}

// objcache.Free
func (ev *eventType) Free() {
	ev.argObject.release()
	ev.argUnlObj.release()
	ev.cud.release()
	if ev.buffer != nil {
		bytespool.Put(ev.buffer)
		ev.buffer = nil
	}
}

// istructs.IDbEvent.QName
func (ev *eventType) QName() appdef.QName {
	if ev.name == istructs.QNameForCorruptedData {
		return istructs.QNameForCorruptedData
	}

	if ev.valid() {
		return ev.name
	}

	return istructs.QNameForError
}

// istructs.IAbstractEvent.RegisteredAt
func (ev *eventType) RegisteredAt() istructs.UnixMilli {
	return ev.regTime
}

// istructs.IPLogEvent.Release and IWLogEvent.Release
func (ev *eventType) Release() {
	// Free() will called through a RefCounter.Release() then reference counter decrease zero
	ev.RefCounter.Release()
}

// # Return event name, such as `event «sys.CUD»` or `event «test.ODocument»`
func (ev *eventType) String() string {
	return fmt.Sprintf("event «%v»", ev.name)
}

// istructs.IAbstractEvent.Synced
func (ev *eventType) Synced() bool {
	return ev.sync
}

// istructs.IAbstractEvent.DeviceID
func (ev *eventType) DeviceID() istructs.ConnectedDeviceID {
	return ev.device
}

// istructs.IAbstractEvent.SyncedAt
func (ev *eventType) SyncedAt() istructs.UnixMilli {
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

// cudType implements event cud member
//
// # Implements:
//
//	— istructs.ICUD
type cudType struct {
	appCfg  *AppConfigType
	creates []*recordType
	updates map[istructs.RecordID]*updateRecType
}

func makeCUD(appCfg *AppConfigType) cudType {
	return cudType{
		appCfg:  appCfg,
		creates: make([]*recordType, 0),
		updates: make(map[istructs.RecordID]*updateRecType),
	}
}

// applyRecs call store callback func for each record
func (cud *cudType) applyRecs(load, store recordFunc) (err error) {

	for _, rec := range cud.creates {
		if err = store(rec); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if rec.originRec.empty() {
			// this case reread event from PLog after restart.
			// It is necessary to:
			//	- load the existing record from the storage and
			// 	- rebuild the result with changes
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

// empty return is all members is empty
func (cud *cudType) empty() bool {
	return (len(cud.creates) == 0) && (len(cud.updates) == 0)
}

// enumRecs: enumerates changes as IRecords
func (cud *cudType) enumRecs(cb func(rec istructs.ICUDRow) bool) {
	for _, rec := range cud.creates {
		if !cb(rec) {
			return
		}
	}

	for _, rec := range cud.updates {
		if !cb(&rec.changes) {
			return
		}
	}
}

// newIDsPlanType is type for ID regeneration plan. Key is raw ID, value is storage ID
type newIDsPlanType map[istructs.RecordID]istructs.RecordID

// regenerateIDsPlan creates new ID regeneration plan
func (cud *cudType) regenerateIDsPlan(generator istructs.IIDGenerator) (newIDs newIDsPlanType, err error) {
	plan := make(newIDsPlanType)
	for _, rec := range cud.creates {
		id := rec.ID()
		if !id.IsRaw() {
			// storage IDs are allowed for sync events
			generator.UpdateOnSync(id, rec.typ)
			continue
		}

		var storeID istructs.RecordID

		if singleton, ok := rec.typ.(appdef.ISingleton); ok && singleton.Singleton() {
			if storeID, err = cud.appCfg.singletons.ID(rec.QName()); err != nil {
				return nil, err
			}
		} else {
			if storeID, err = generator.NextID(id, rec.typ); err != nil {
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

	for name, value := range rec.RecordIDs(false) {
		if !value.IsRaw() {
			continue
		}
		if id, ok := newIDs[value]; ok {
			rec.PutRecordID(name, id)
			changes = true
		}
	}
	if changes {
		// rebuild record to apply changes to dyno-buffer
		err = rec.build()
	}
	return err
}

// regenerateIDsInUpdateRecord regenerates ID in single CUD update record changes using specified plan
func regenerateIDsInUpdateRecord(rec *updateRecType, newIDs newIDsPlanType) (err error) {
	changes := false

	for name, value := range rec.changes.RecordIDs(false) {
		if !value.IsRaw() {
			continue
		}
		if id, ok := newIDs[value]; ok {
			rec.changes.PutRecordID(name, id)
			changes = true
		}
	}

	if changes {
		// rebuild record (changes and result) to apply changes to dyno-buffer
		err = rec.build()
	}
	return err
}

// Regenerates all raw IDs to storage IDs
func (cud *cudType) regenerateIDs(generator istructs.IIDGenerator) error {

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

// Returns dynobuffers for all creates and updates to pool
func (cud *cudType) release() {
	for _, c := range cud.creates {
		c.release()
	}
	for _, u := range cud.updates {
		u.release()
	}
}

// istructs.ICUD.Create
func (cud *cudType) Create(qName appdef.QName) istructs.IRowWriter {
	rec := newRecord(cud.appCfg)
	rec.isNew = true
	rec.setQName(qName)

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
		originRec: makeRecord(appCfg),
		changes:   makeRecord(appCfg),
		result:    makeRecord(appCfg),
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

	if upd.changes.QName() == appdef.NullQName {
		return nil
	}

	if err = upd.changes.build(); err != nil {
		return err
	}

	if upd.originRec.ID() != upd.changes.ID() {
		return ErrUnableToUpdateSystemField(upd.originRec, appdef.SystemField_ID)
	}
	if (upd.changes.Parent() != istructs.NullRecordID) && (upd.changes.Parent() != upd.originRec.Parent()) {
		return ErrUnableToUpdateSystemField(upd.originRec, appdef.SystemField_ParentID)
	}
	if (upd.changes.Container() != "") && (upd.changes.Container() != upd.originRec.Container()) {
		return ErrUnableToUpdateSystemField(upd.originRec, appdef.SystemField_Container)
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
	for n := range upd.changes.nils {
		upd.result.dyB.Set(n, nil)
		userChanges = true
	}

	if userChanges {
		err = upd.result.build()
	}

	return err
}

// Return dynobuffers of all recs (origin, changes and result) to pool
func (upd *updateRecType) release() {
	upd.originRec.release()
	upd.changes.release()
	upd.result.release()
}

func (upd *updateRecType) String() string {
	return fmt.Sprint("updated", upd.changes)
}

// # Implements object structure
//
// # Implements:
//
//   - istructs.IObjectBuilder
//   - istructs.IObject,
type objectType struct {
	recordType
	parent *objectType
	child  []*objectType
}

func makeObject(appCfg *AppConfigType, qn appdef.QName, parent *objectType) objectType {
	o := objectType{
		recordType: makeRecord(appCfg),
		parent:     parent,
		child:      make([]*objectType, 0),
	}
	if qn != appdef.NullQName {
		o.setQName(qn)
	}
	return o
}

func newObject(appCfg *AppConfigType, qn appdef.QName, parent *objectType) *objectType {
	o := makeObject(appCfg, qn, parent)
	return &o
}

// enumerates all children
func (o *objectType) allChildren(cb func(*objectType)) {
	for _, c := range o.child {
		cb(c)
	}
}

// Builds object with children recursive
func (o *objectType) build() (err error) {
	if len(o.child) > math.MaxUint16 {
		// because len(o.child) will be stored as uint16, see [storeObject]
		return validateErrorf(ECode_TooManyChildren, "children number must not be more than %d", math.MaxUint16)
	}
	return o.forEach(func(c *objectType) error {
		return c.rowType.build()
	})
}

// Clears object with children recursive
func (o *objectType) clear() {
	o.recordType.clear()
	o.child = make([]*objectType, 0)
}

// forEach applies cb function to element and all it children recursive
func (o *objectType) forEach(cb func(c *objectType) error) (err error) {
	if err = cb(o); err == nil {
		for _, e := range o.child {
			if err = e.forEach(cb); err != nil {
				break
			}
		}
	}
	return err
}

// Returns is object has document type kind
func (o *objectType) isDocument() bool {
	kind := o.typ.Kind()
	return (kind == appdef.TypeKind_GDoc) ||
		(kind == appdef.TypeKind_CDoc) ||
		(kind == appdef.TypeKind_ODoc) ||
		(kind == appdef.TypeKind_WDoc)
}

// maskValues masks object row values with children recursive
func (o *objectType) maskValues() {
	o.rowType.maskValues()

	for _, c := range o.child {
		c.maskValues()
	}
}

// regenerateIDs regenerates record IDs in object and children recursive.
// If some child record ID reference (e.c. «sys.Parent» fields) refers to regenerated parent ID fields, this replaced too.
func (o *objectType) regenerateIDs(generator istructs.IIDGenerator) (err error) {
	newIDs := make(newIDsPlanType)

	err = o.forEach(
		func(c *objectType) error {
			if id := c.ID(); id.IsRaw() {
				storeID, err := generator.NextID(id, c.typ)
				if err != nil {
					return err
				}
				c.setID(storeID)
				newIDs[id] = storeID
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = o.forEach(
		func(c *objectType) (err error) {
			if id := c.Parent(); id.IsRaw() {
				c.setParent(newIDs[id])
			}

			changes := false
			for name, id := range c.RecordIDs(false) {
				if id.IsRaw() {
					c.PutRecordID(name, newIDs[id])
					changes = true
				}
			}
			if changes {
				// rebuild object to apply changes in dyno-buffer
				err = c.build()
			}
			return err
		})

	return err
}

// Return dynobuffer to pool for object with children recursive
func (o *objectType) release() {
	o.recordType.release()
	for _, c := range o.child {
		c.release()
	}
}

// # Implements istructs.IObjectBuilder.Build()
//
// Builds and returns object or document.
//
//	If builded object type is not found in appdef then returns error.
//	If builded object type is not object or document then returns error.
//	If builded object is not valid then returns validation error.
func (o *objectType) Build() (istructs.IObject, error) {
	if err := o.build(); err != nil {
		return nil, err
	}
	if o.QName() == appdef.NullQName {
		return nil, ErrNameMissed("object builder has empty type name")
	}
	if t := o.typ.Kind(); (t != appdef.TypeKind_Object) &&
		(t != appdef.TypeKind_ODoc) &&
		(t != appdef.TypeKind_GDoc) &&
		(t != appdef.TypeKind_CDoc) &&
		(t != appdef.TypeKind_WDoc) {
		return nil, ErrUnexpectedType("%v, should be object or document", o)
	}
	if _, err := validateObjectIDs(o, false); err != nil {
		return nil, err
	}
	if err := validateObject(o); err != nil {
		return nil, err
	}

	return o, nil
}

// istructs.IObjectBuilder.ChildBuilder
func (o *objectType) ChildBuilder(containerName string) istructs.IObjectBuilder {
	c := newObject(o.appCfg, appdef.NullQName, o)
	o.child = append(o.child, c)
	if o.QName() != appdef.NullQName {
		if cont := o.typ.(appdef.IContainers).Container(containerName); cont != nil {
			c.setQName(cont.QName())
			if c.QName() != appdef.NullQName {
				if o.ID() != istructs.NullRecordID {
					c.setParent(o.ID())
				}
				c.setContainer(containerName)
			}
		}
	}
	return c
}

// istructs.IObject.Children
func (o *objectType) Children(container ...string) func(func(istructs.IObject) bool) {
	c := make(map[string]bool, len(container))
	for _, cont := range container {
		c[cont] = true
	}
	return func(cb func(istructs.IObject) bool) {
		for _, child := range o.child {
			if len(c) == 0 || c[child.Container()] {
				if !cb(child) {
					break
				}
			}
		}
	}
}

// istructs.IObject.Containers
func (o *objectType) Containers(cb func(string) bool) {
	duplicates := make(map[string]bool, len(o.child))
	for _, c := range o.child {
		name := c.Container()
		if duplicates[name] {
			continue
		}
		if !cb(name) {
			break
		}
		duplicates[name] = true
	}
}

// istructs.IObjectBuilder.FillFromJSON
func (o *objectType) FillFromJSON(data map[string]any) {
	for n, v := range data {
		switch fv := v.(type) {
		case nil:
		case float64:
			o.PutFloat64(n, fv)
		case istructs.RecordID:
			o.PutRecordID(n, fv)
		case int32:
			o.PutInt32(n, fv)
		case int64:
			o.PutInt64(n, fv)
		case json.Number:
			o.PutNumber(n, fv)
		case string:
			o.PutChars(n, fv)
		case bool:
			o.PutBool(n, fv)
		case []interface{}:
			// e.g. "order_item": [<2 children>]
			cont := o.typ.(appdef.IContainers).Container(n)
			if cont == nil {
				o.collectError(ErrContainerNotFound(n, o.typ))
				continue
			}
			for i, val := range fv {
				childData, ok := val.(map[string]any)
				if !ok {
					o.collectErrorf("%v: invalid type «%T» in JSON for child «%s[%d]», expected «map[string]any»: %w", o, val, n, i, ErrWrongTypeError)
					break
				}
				c := o.ChildBuilder(n)
				c.FillFromJSON(childData)
			}
		default:
			o.collectError(ErrWrongType(`%#T for field "%s" with value %v`, v, n, v))
		}
	}
}

// istructs.IObject.QName()
func (o *objectType) QName() appdef.QName {
	return o.recordType.QName()
}

// istructs.IObject.AsRecord()
func (o *objectType) AsRecord() istructs.IRecord {
	return o
}

// Implements interfaces:
//
//	— istructs.IEventError
type eventErrorType struct {
	validEvent bool
	errStr     string
	qName      appdef.QName
	bytes      []byte
}

func makeEventError() eventErrorType {
	return eventErrorType{
		validEvent: true,
		qName:      appdef.NullQName,
	}
}

// Sets event build error
func (e *eventErrorType) setError(event *eventType, err error) {
	if err == nil {
		e.validEvent = true
		e.errStr = ""
		e.qName = appdef.NullQName
		e.bytes = nil
	} else {
		e.validEvent = false
		e.errStr = err.Error()
		e.qName = event.name
		e.bytes = utils.CopyBytes(event.rawBytes)
	}
}

// istructs.IEventError.ErrStr
func (e *eventErrorType) ErrStr() string {
	return e.errStr
}

// istructs.IEventError.QNameFromParams
func (e *eventErrorType) QNameFromParams() appdef.QName {
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
