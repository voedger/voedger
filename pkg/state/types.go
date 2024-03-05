/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

type PartitionIDFunc func() istructs.PartitionID
type WSIDFunc func() istructs.WSID
type N10nFunc func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset)
type AppStructsFunc func() istructs.IAppStructs
type CUDFunc func() istructs.ICUD
type CmdResultBuilderFunc func() istructs.IObjectBuilder
type PrincipalsFunc func() []iauthnz.Principal
type TokenFunc func() string
type PLogEventFunc func() istructs.IPLogEvent
type ArgFunc func() istructs.IObject
type UnloggedArgFunc func() istructs.IObject
type CommandProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, intentsLimit int, cmdResultBuilderFunc CmdResultBuilderFunc, argFunc ArgFunc, unloggedArgFunc UnloggedArgFunc) IHostState
type SyncActualizerStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit int) IHostState
type QueryProcessorStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, argFunc ArgFunc) IHostState
type AsyncActualizerStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit, bundlesLimit int,
	opts ...ActualizerStateOptFunc) IBundledHostState

type eventsFunc func() istructs.IEvents
type viewRecordsFunc func() istructs.IViewRecords
type recordsFunc func() istructs.IRecords
type appDefFunc func() appdef.IAppDef

type ApplyBatchItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
}

type GetBatchItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValue
}

type keyBuilder struct {
	data    map[string]interface{}
	storage appdef.QName
	entity  appdef.QName
}

func newKeyBuilder(storage, entity appdef.QName) *keyBuilder {
	return &keyBuilder{
		data:    make(map[string]interface{}),
		storage: storage,
		entity:  entity,
	}
}

func (b *keyBuilder) Storage() appdef.QName                            { return b.storage }
func (b *keyBuilder) Entity() appdef.QName                             { return b.entity }
func (b *keyBuilder) PutInt32(name string, value int32)                { b.data[name] = value }
func (b *keyBuilder) PutInt64(name string, value int64)                { b.data[name] = value }
func (b *keyBuilder) PutFloat32(name string, value float32)            { b.data[name] = value }
func (b *keyBuilder) PutFloat64(name string, value float64)            { b.data[name] = value }
func (b *keyBuilder) PutBytes(name string, value []byte)               { b.data[name] = value }
func (b *keyBuilder) PutString(name string, value string)              { b.data[name] = value }
func (b *keyBuilder) PutQName(name string, value appdef.QName)         { b.data[name] = value }
func (b *keyBuilder) PutBool(name string, value bool)                  { b.data[name] = value }
func (b *keyBuilder) PutRecordID(name string, value istructs.RecordID) { b.data[name] = value }
func (b *keyBuilder) PutNumber(string, float64)                        { panic(ErrNotSupported) }
func (b *keyBuilder) PutChars(string, string)                          { panic(ErrNotSupported) }
func (b *keyBuilder) PutFromJSON(j map[string]any)                     { maps.Copy(b.data, j) }
func (b *keyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) }
func (b *keyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) }
func (b *keyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*keyBuilder)
	if !ok {
		return false
	}
	if b.storage != kb.storage {
		return false
	}
	if b.entity != kb.entity {
		return false
	}
	if !maps.Equal(b.data, kb.data) {
		return false
	}
	return true
}

type logKeyBuilder struct {
	istructs.IStateKeyBuilder
	offset istructs.Offset
	count  int
}

func (b *logKeyBuilder) PutInt64(name string, value int64) {
	switch name {
	case Field_Offset:
		b.offset = istructs.Offset(value)
	case Field_Count:
		b.count = int(value)
	}
}

type pLogKeyBuilder struct {
	logKeyBuilder
	partitionID istructs.PartitionID
}

func (b *pLogKeyBuilder) Storage() appdef.QName {
	return PLog
}

func (b *pLogKeyBuilder) String() string {
	return fmt.Sprintf("plog partitionID - %d, offset - %d, count - %d", b.partitionID, b.offset, b.count)
}

func (b *pLogKeyBuilder) PutInt32(name string, value int32) {
	if name == Field_PartitionID {
		b.partitionID = istructs.PartitionID(value)
	}
}

type wLogKeyBuilder struct {
	logKeyBuilder
	wsid istructs.WSID
}

func (b *wLogKeyBuilder) Storage() appdef.QName {
	return WLog
}

func (b *wLogKeyBuilder) String() string {
	return fmt.Sprintf("wlog wsid - %d, offset - %d, count - %d", b.wsid, b.offset, b.count)
}

func (b *wLogKeyBuilder) PutInt64(name string, value int64) {
	b.logKeyBuilder.PutInt64(name, value)
	if name == Field_WSID {
		b.wsid = istructs.WSID(value)
	}
}

type recordsKeyBuilder struct {
	istructs.IStateKeyBuilder
	id        istructs.RecordID
	singleton appdef.QName
	wsid      istructs.WSID
	entity    appdef.QName
}

func (b *recordsKeyBuilder) Storage() appdef.QName {
	return Record
}

func (b *recordsKeyBuilder) String() string {
	sb := strings.Builder{}
	_, _ = sb.WriteString(fmt.Sprintf("- %T", b))
	if b.id != istructs.NullRecordID {
		_, _ = sb.WriteString(fmt.Sprintf(", ID - %d", b.id))
	}
	if b.singleton != appdef.NullQName {
		_, _ = sb.WriteString(fmt.Sprintf(", singleton - %s", b.singleton))
	}
	_, _ = sb.WriteString(fmt.Sprintf(", WSID - %d", b.wsid))
	return sb.String()
}

func (b *recordsKeyBuilder) PutInt64(name string, value int64) {
	if name == Field_WSID {
		b.wsid = istructs.WSID(value)
		return
	}
	// TODO ???
	panic(name)
}

func (b *recordsKeyBuilder) PutRecordID(name string, value istructs.RecordID) {
	if name == Field_ID {
		b.id = value
		return
	}
	// TODO ???
	panic(name)
}

func (b *recordsKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == Field_Singleton {
		b.singleton = value
		return
	}
	// TODO ???
	panic(name)
}

type recordsValueBuilder struct {
	istructs.IStateValueBuilder
	rw istructs.IRowWriter
}

func (b *recordsValueBuilder) PutInt32(name string, value int32)        { b.rw.PutInt32(name, value) }
func (b *recordsValueBuilder) PutInt64(name string, value int64)        { b.rw.PutInt64(name, value) }
func (b *recordsValueBuilder) PutBytes(name string, value []byte)       { b.rw.PutBytes(name, value) }
func (b *recordsValueBuilder) PutString(name, value string)             { b.rw.PutString(name, value) }
func (b *recordsValueBuilder) PutBool(name string, value bool)          { b.rw.PutBool(name, value) }
func (b *recordsValueBuilder) PutChars(name string, value string)       { b.rw.PutChars(name, value) }
func (b *recordsValueBuilder) PutFloat32(name string, value float32)    { b.rw.PutFloat32(name, value) }
func (b *recordsValueBuilder) PutFloat64(name string, value float64)    { b.rw.PutFloat64(name, value) }
func (b *recordsValueBuilder) PutQName(name string, value appdef.QName) { b.rw.PutQName(name, value) }
func (b *recordsValueBuilder) PutNumber(name string, value float64)     { b.rw.PutNumber(name, value) }
func (b *recordsValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	b.rw.PutRecordID(name, value)
}

type viewKeyBuilder struct {
	istructs.IKeyBuilder
	wsid istructs.WSID
	view appdef.QName
}

func (b *viewKeyBuilder) PutInt64(name string, value int64) {
	if name == Field_WSID {
		b.wsid = istructs.WSID(value)
		return
	}
	b.IKeyBuilder.PutInt64(name, value)
}
func (b *viewKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.wsid = istructs.NullWSID
		b.view = value
	}
	b.IKeyBuilder.PutQName(name, value)
}
func (b *viewKeyBuilder) Entity() appdef.QName {
	return b.view
}
func (b *viewKeyBuilder) Storage() appdef.QName {
	return View
}
func (b *viewKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*viewKeyBuilder)
	if !ok {
		return false
	}
	if b.wsid != kb.wsid {
		return false
	}
	if b.view != kb.view {
		return false
	}
	return b.IKeyBuilder.Equals(kb.IKeyBuilder)
}

type viewValueBuilder struct {
	istructs.IValueBuilder
	offset istructs.Offset
	entity appdef.QName
}

func (b *viewValueBuilder) PutInt64(name string, value int64) {
	if name == ColOffset {
		b.offset = istructs.Offset(value)
	}
	b.IValueBuilder.PutInt64(name, value)
}
func (b *viewValueBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.offset = istructs.NullOffset
	}
	b.IValueBuilder.PutQName(name, value)
}
func (b *viewValueBuilder) Build() istructs.IValue {
	return b.IValueBuilder.Build()
}

func (b *viewValueBuilder) BuildValue() istructs.IStateValue {
	return &viewValue{
		value: b.Build(),
	}
}

type recordsValue struct {
	baseStateValue
	record istructs.IRecord
}

func (v *recordsValue) AsInt32(name string) int32        { return v.record.AsInt32(name) }
func (v *recordsValue) AsInt64(name string) int64        { return v.record.AsInt64(name) }
func (v *recordsValue) AsFloat32(name string) float32    { return v.record.AsFloat32(name) }
func (v *recordsValue) AsFloat64(name string) float64    { return v.record.AsFloat64(name) }
func (v *recordsValue) AsBytes(name string) []byte       { return v.record.AsBytes(name) }
func (v *recordsValue) AsString(name string) string      { return v.record.AsString(name) }
func (v *recordsValue) AsQName(name string) appdef.QName { return v.record.AsQName(name) }
func (v *recordsValue) AsBool(name string) bool          { return v.record.AsBool(name) }
func (v *recordsValue) AsRecordID(name string) istructs.RecordID {
	return v.record.AsRecordID(name)
}
func (v *recordsValue) AsRecord(string) (record istructs.IRecord) { return v.record }
func (v *recordsValue) FieldNames(cb func(fieldName string)) {
	v.record.FieldNames(cb)
}

type objectArrayContainerValue struct {
	baseStateValue
	object    istructs.IObject
	container string
}

func (v *objectArrayContainerValue) GetAsString(int) string      { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsBytes(int) []byte       { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsInt32(int) int32        { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsInt64(int) int64        { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsFloat32(int) float32    { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsFloat64(int) float64    { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsQName(int) appdef.QName { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsBool(int) bool          { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsValue(i int) (result istructs.IStateValue) {
	index := 0
	v.object.Children(v.container, func(o istructs.IObject) {
		if index == i {
			result = &objectValue{object: o}
		}
		index++
	})
	if result == nil {
		panic(errIndexOutOfBounds(i))
	}
	return
}
func (v *objectArrayContainerValue) Length() int {
	var result int
	v.object.Children(v.container, func(i istructs.IObject) {
		result++
	})
	return result
}

type objectValue struct {
	baseStateValue
	object istructs.IObject
}

func (v *objectValue) AsInt32(name string) int32                { return v.object.AsInt32(name) }
func (v *objectValue) AsInt64(name string) int64                { return v.object.AsInt64(name) }
func (v *objectValue) AsFloat32(name string) float32            { return v.object.AsFloat32(name) }
func (v *objectValue) AsFloat64(name string) float64            { return v.object.AsFloat64(name) }
func (v *objectValue) AsBytes(name string) []byte               { return v.object.AsBytes(name) }
func (v *objectValue) AsString(name string) string              { return v.object.AsString(name) }
func (v *objectValue) AsQName(name string) appdef.QName         { return v.object.AsQName(name) }
func (v *objectValue) AsBool(name string) bool                  { return v.object.AsBool(name) }
func (v *objectValue) AsRecordID(name string) istructs.RecordID { return v.object.AsRecordID(name) }
func (v *objectValue) RecordIDs(includeNulls bool, cb func(string, istructs.RecordID)) {
	v.object.RecordIDs(includeNulls, cb)
}
func (v *objectValue) FieldNames(cb func(string)) { v.object.FieldNames(cb) }
func (v *objectValue) AsValue(name string) (result istructs.IStateValue) {
	v.object.Containers(func(name string) {
		result = &objectArrayContainerValue{
			object:    v.object,
			container: name,
		}
	})
	if result == nil {
		panic(errUndefined(name))
	}
	return
}

type pLogValue struct {
	baseStateValue
	event  istructs.IPLogEvent
	offset int64
}

func (v *pLogValue) AsInt64(name string) int64 {
	switch name {
	case Field_WLogOffset:
		return int64(v.event.WLogOffset())
	case Field_Workspace:
		return int64(v.event.Workspace())
	case Field_RegisteredAt:
		return int64(v.event.RegisteredAt())
	case Field_DeviceID:
		return int64(v.event.DeviceID())
	case Field_SyncedAt:
		return int64(v.event.SyncedAt())
	case Field_Offset:
		return v.offset
	}
	panic(errUndefined(name))
}
func (v *pLogValue) AsBool(name string) bool {
	if name == Field_Synced {
		return v.event.Synced()
	}
	panic(errUndefined(name))
}
func (v *pLogValue) AsRecord(string) istructs.IRecord {
	return v.event.ArgumentObject().AsRecord()
}
func (v *pLogValue) AsQName(name string) appdef.QName {
	if name == Field_QName {
		return v.event.QName()
	}
	panic(errUndefined(name))
}
func (v *pLogValue) AsEvent(string) istructs.IDbEvent { return v.event }
func (v *pLogValue) AsValue(name string) istructs.IStateValue {
	if name == Field_CUDs {
		sv := &cudsValue{}
		v.event.CUDs(func(rec istructs.ICUDRow) {
			sv.cuds = append(sv.cuds, rec)
		})
		return sv
	}
	if name == Field_Error {
		return &eventErrorValue{error: v.event.Error()}
	}
	if name == Field_ArgumentObject {
		arg := v.event.ArgumentObject()
		if arg == nil {
			return nil
		}
		return &objectValue{object: arg}
	}
	panic(errUndefined(name))
}

type wLogValue struct {
	baseStateValue
	event  istructs.IWLogEvent
	offset int64
}

func (v *wLogValue) AsInt64(name string) int64 {
	switch name {
	case Field_RegisteredAt:
		return int64(v.event.RegisteredAt())
	case Field_DeviceID:
		return int64(v.event.DeviceID())
	case Field_SyncedAt:
		return int64(v.event.SyncedAt())
	case Field_Offset:
		return v.offset
	default:
		return 0
	}
}
func (v *wLogValue) AsBool(_ string) bool          { return v.event.Synced() }
func (v *wLogValue) AsQName(_ string) appdef.QName { return v.event.QName() }
func (v *wLogValue) AsEvent(_ string) (event istructs.IDbEvent) {
	return v.event
}
func (v *wLogValue) AsRecord(_ string) (record istructs.IRecord) {
	return v.event.ArgumentObject().AsRecord()
}
func (v *wLogValue) AsValue(name string) istructs.IStateValue {
	if name != Field_CUDs {
		panic(ErrNotSupported)
	}
	sv := &cudsValue{}
	v.event.CUDs(func(rec istructs.ICUDRow) {
		sv.cuds = append(sv.cuds, rec)
	})
	return sv
}

type sendMailKeyBuilder struct {
	*keyBuilder
	to  []string
	cc  []string
	bcc []string
}

func (b *sendMailKeyBuilder) PutString(name string, value string) {
	switch name {
	case Field_To:
		b.to = append(b.to, value)
	case Field_CC:
		b.cc = append(b.cc, value)
	case Field_BCC:
		b.bcc = append(b.bcc, value)
	default:
		b.keyBuilder.PutString(name, value)
	}
}

type httpKeyBuilder struct {
	*keyBuilder
	headers map[string]string
}

func newHttpKeyBuilder() *httpKeyBuilder {
	return &httpKeyBuilder{
		keyBuilder: newKeyBuilder(Http, appdef.NullQName),
		headers:    make(map[string]string),
	}
}

func (b *httpKeyBuilder) PutString(name string, value string) {
	switch name {
	case Field_Header:
		trim := func(v string) string { return strings.Trim(v, " \n\r\t") }
		ss := strings.SplitN(value, ":", 2)
		b.headers[trim(ss[0])] = trim(ss[1])
	default:
		b.keyBuilder.PutString(name, value)
	}
}

func (b *httpKeyBuilder) method() string {
	if v, ok := b.keyBuilder.data[Field_Method]; ok {
		return v.(string)
	}
	return http.MethodGet
}
func (b *httpKeyBuilder) url() string {
	if v, ok := b.keyBuilder.data[Field_Url]; ok {
		return v.(string)
	}
	panic(fmt.Errorf("'url': %w", ErrNotFound))
}
func (b *httpKeyBuilder) body() io.Reader {
	if v, ok := b.keyBuilder.data[Field_Body]; ok {
		return bytes.NewReader(v.([]byte))
	}
	return nil
}
func (b *httpKeyBuilder) timeout() time.Duration {
	if v, ok := b.keyBuilder.data[Field_HTTPClientTimeoutMilliseconds]; ok {
		t := v.(int64)
		return time.Duration(t) * time.Millisecond
	}
	return defaultHTTPClientTimeout
}
func (b *httpKeyBuilder) String() string {
	ss := make([]string, 0, httpStorageKeyBuilderStringerSliceCap)
	ss = append(ss, b.method())
	ss = append(ss, b.url())
	if v, ok := b.keyBuilder.data[Field_Body]; ok {
		ss = append(ss, string(v.([]byte)))
	}
	return strings.Join(ss, " ")
}

type httpValue struct {
	istructs.IStateValue
	body       []byte
	header     map[string][]string
	statusCode int
}

func (v *httpValue) AsBytes(string) []byte { return v.body }
func (v *httpValue) AsInt32(string) int32  { return int32(v.statusCode) }
func (v *httpValue) AsString(name string) string {
	if name == Field_Header {
		var res strings.Builder
		for k, v := range v.header {
			if len(v) > 0 {
				if res.Len() > 0 {
					res.WriteString("\n")
				}
				res.WriteString(fmt.Sprintf("%s: %s", k, v[0])) // FIXME: len(v)>2 ?
			}
		}
		return res.String()
	}
	return string(v.body)
}

type appSecretValue struct {
	baseStateValue
	content string
}

func (v *appSecretValue) AsString(string) string { return v.content }

type n10n struct {
	wsid istructs.WSID
	view appdef.QName
}

type bundle interface {
	put(key istructs.IStateKeyBuilder, value ApplyBatchItem)
	get(key istructs.IStateKeyBuilder) (value ApplyBatchItem, ok bool)
	containsKeysForSameEntity(key istructs.IStateKeyBuilder) bool
	values() (values []ApplyBatchItem)
	size() (size int)
	clear()
}

type pair struct {
	key   istructs.IStateKeyBuilder
	value ApplyBatchItem
}

type bundleImpl struct {
	list *list.List
}

func newBundle() bundle {
	return &bundleImpl{list: list.New()}
}

func (b *bundleImpl) put(key istructs.IStateKeyBuilder, value ApplyBatchItem) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		if el.Value.(*pair).key.Equals(key) {
			el.Value.(*pair).value = value
			return
		}
	}
	b.list.PushBack(&pair{key: key, value: value})
}
func (b *bundleImpl) get(key istructs.IStateKeyBuilder) (value ApplyBatchItem, ok bool) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		if el.Value.(*pair).key.Equals(key) {
			return el.Value.(*pair).value, true
		}
	}
	return emptyApplyBatchItem, false
}
func (b *bundleImpl) containsKeysForSameEntity(key istructs.IStateKeyBuilder) bool {
	var next *list.Element
	for el := b.list.Front(); el != nil; el = next {
		next = el.Next()
		if el.Value.(*pair).key.Entity() == key.Entity() {
			return true
		}
	}
	return false
}
func (b *bundleImpl) values() (values []ApplyBatchItem) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		values = append(values, el.Value.(*pair).value)
	}
	return
}
func (b *bundleImpl) size() (size int) { return b.list.Len() }
func (b *bundleImpl) clear()           { b.list = list.New() }

type key struct {
	istructs.IKey
	data map[string]interface{}
}

func (k *key) AsInt64(name string) int64 { return k.data[name].(int64) }

type requestSubjectValue struct {
	baseStateValue
	kind        int32
	profileWSID int64
	name        string
	token       string
}

func (v *requestSubjectValue) AsInt64(name string) int64 {
	switch name {
	case Field_ProfileWSID:
		return v.profileWSID
	default:
		return 0
	}
}
func (v *requestSubjectValue) AsInt32(name string) int32 {
	switch name {
	case Field_Kind:
		return v.kind
	default:
		return 0
	}
}
func (v *requestSubjectValue) AsString(name string) string {
	switch name {
	case Field_Name:
		return v.name
	case Field_Token:
		return v.token
	default:
		return ""
	}
}

type viewValue struct {
	baseStateValue
	value istructs.IValue
}

func (v *viewValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *viewValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *viewValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *viewValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *viewValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *viewValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *viewValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *viewValue) AsBool(name string) bool          { return v.value.AsBool(name) }
func (v *viewValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}
func (v *viewValue) AsRecord(name string) istructs.IRecord {
	return v.value.AsRecord(name)
}

type eventErrorValue struct {
	istructs.IStateValue
	error istructs.IEventError
}

func (v *eventErrorValue) AsString(name string) string {
	if name == Field_ErrStr {
		return v.error.ErrStr()
	}
	panic(ErrNotSupported)
}

func (v *eventErrorValue) AsBool(name string) bool {
	if name == Field_ValidEvent {
		return v.error.ValidEvent()
	}
	panic(ErrNotSupported)
}

func (v *eventErrorValue) AsQName(name string) appdef.QName {
	if name == Field_QNameFromParams {
		return v.error.QNameFromParams()
	}
	panic(ErrNotSupported)
}

type cudsValue struct {
	istructs.IStateValue
	cuds []istructs.ICUDRow
}

func (v *cudsValue) Length() int { return len(v.cuds) }
func (v *cudsValue) GetAsValue(index int) istructs.IStateValue {
	return &cudRowValue{value: v.cuds[index]}
}

type cudRowValue struct {
	baseStateValue
	value istructs.ICUDRow
}

func (v *cudRowValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *cudRowValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *cudRowValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *cudRowValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *cudRowValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *cudRowValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *cudRowValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *cudRowValue) AsBool(name string) bool {
	if name == Field_IsNew {
		return v.value.IsNew()
	}
	return v.value.AsBool(name)
}
func (v *cudRowValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}

type baseStateValue struct{}

func (v *baseStateValue) AsInt32(string) int32                            { panic(errNotImplemented) }
func (v *baseStateValue) AsInt64(string) int64                            { panic(errNotImplemented) }
func (v *baseStateValue) AsFloat32(string) float32                        { panic(errNotImplemented) }
func (v *baseStateValue) AsFloat64(string) float64                        { panic(errNotImplemented) }
func (v *baseStateValue) AsBytes(string) []byte                           { panic(errNotImplemented) }
func (v *baseStateValue) AsString(string) string                          { panic(errNotImplemented) }
func (v *baseStateValue) AsQName(string) appdef.QName                     { panic(errNotImplemented) }
func (v *baseStateValue) AsBool(string) bool                              { panic(errNotImplemented) }
func (v *baseStateValue) AsRecordID(string) istructs.RecordID             { panic(errNotImplemented) }
func (v *baseStateValue) RecordIDs(bool, func(string, istructs.RecordID)) { panic(errNotImplemented) }
func (v *baseStateValue) FieldNames(func(string))                         { panic(errNotImplemented) }
func (v *baseStateValue) AsRecord(string) istructs.IRecord                { panic(errNotImplemented) }
func (v *baseStateValue) AsEvent(string) istructs.IDbEvent                { panic(errNotImplemented) }
func (v *baseStateValue) Length() int                                     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsString(int) string                          { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBytes(int) []byte                           { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt32(int) int32                            { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt64(int) int64                            { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat32(int) float32                        { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat64(int) float64                        { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsQName(int) appdef.QName                     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBool(int) bool                              { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsValue(int) istructs.IStateValue {
	panic(errFieldByIndexIsNotAnObjectOrArray)
}
func (v *baseStateValue) AsValue(string) istructs.IStateValue {
	panic(errFieldByNameIsNotAnObjectOrArray)
}

type resultKeyBuilder struct {
	*keyBuilder
}

func newResultKeyBuilder() *resultKeyBuilder {
	return &resultKeyBuilder{
		&keyBuilder{
			storage: Result,
		},
	}
}

type resultValueBuilder struct {
	istructs.IStateValueBuilder
	resultBuilder istructs.IObjectBuilder
}

func (c *resultValueBuilder) PutInt32(name string, value int32) {
	c.resultBuilder.PutInt32(name, value)
}

func (c *resultValueBuilder) PutInt64(name string, value int64) {
	c.resultBuilder.PutInt64(name, value)
}
func (c *resultValueBuilder) PutBytes(name string, value []byte) {
	c.resultBuilder.PutBytes(name, value)
}
func (c *resultValueBuilder) PutString(name, value string) {
	c.resultBuilder.PutString(name, value)
}
func (c *resultValueBuilder) PutBool(name string, value bool) {
	c.resultBuilder.PutBool(name, value)
}
func (c *resultValueBuilder) PutChars(name string, value string) {
	c.resultBuilder.PutChars(name, value)
}
func (c *resultValueBuilder) PutFloat32(name string, value float32) {
	c.resultBuilder.PutFloat32(name, value)
}
func (c *resultValueBuilder) PutFloat64(name string, value float64) {
	c.resultBuilder.PutFloat64(name, value)
}
func (c *resultValueBuilder) PutQName(name string, value appdef.QName) {
	c.resultBuilder.PutQName(name, value)
}
func (c *resultValueBuilder) PutNumber(name string, value float64) {
	c.resultBuilder.PutNumber(name, value)
}
func (c *resultValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	c.resultBuilder.PutRecordID(name, value)
}
