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
type CommandProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, intentsLimit int, cmdResultBuilderFunc CmdResultBuilderFunc) IHostState
type SyncActualizerStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit int) IHostState
type QueryProcessorStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc) IHostState
type AsyncActualizerStateFactory func(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit, bundlesLimit int,
	opts ...ActualizerStateOptFunc) IBundledHostState

type eventsFunc func() istructs.IEvents
type viewRecordsFunc func() istructs.IViewRecords
type recordsFunc func() istructs.IRecords
type appDefFunc func() appdef.IAppDef

type ToJSONOptions struct{ excludedFields map[string]bool }
type ToJSONOption func(opts *ToJSONOptions)
type toJSONFunc func(e istructs.IStateValue, opts ...interface{}) (string, error)

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
	if len(b.data) != len(kb.data) {
		return false
	}
	for k, v := range b.data {
		if v != kb.data[k] {
			return false
		}
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
	//TODO ???
	panic(name)
}

func (b *recordsKeyBuilder) PutRecordID(name string, value istructs.RecordID) {
	if name == Field_ID {
		b.id = value
		return
	}
	//TODO ???
	panic(name)
}

func (b *recordsKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == Field_Singleton {
		b.singleton = value
		return
	}
	//TODO ???
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

type viewRecordsKeyBuilder struct {
	istructs.IKeyBuilder
	wsid istructs.WSID
	view appdef.QName
}

func (b *viewRecordsKeyBuilder) PutInt64(name string, value int64) {
	if name == Field_WSID {
		b.wsid = istructs.WSID(value)
		return
	}
	b.IKeyBuilder.PutInt64(name, value)
}
func (b *viewRecordsKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.wsid = istructs.NullWSID
		b.view = value
	}
	b.IKeyBuilder.PutQName(name, value)
}
func (b *viewRecordsKeyBuilder) Entity() appdef.QName {
	return b.view
}
func (b *viewRecordsKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*viewRecordsKeyBuilder)
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

type viewRecordsValueBuilder struct {
	istructs.IValueBuilder
	offset     istructs.Offset
	toJSONFunc toJSONFunc
	entity     appdef.QName
}

func (b *viewRecordsValueBuilder) PutInt64(name string, value int64) {
	if name == ColOffset {
		b.offset = istructs.Offset(value)
	}
	b.IValueBuilder.PutInt64(name, value)
}
func (b *viewRecordsValueBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.offset = istructs.NullOffset
	}
	b.IValueBuilder.PutQName(name, value)
}
func (b *viewRecordsValueBuilder) Build() istructs.IValue {
	return b.IValueBuilder.Build()
}

func (b *viewRecordsValueBuilder) BuildValue() istructs.IStateValue {
	return &viewRecordsStorageValue{
		value:      b.Build(),
		toJSONFunc: b.toJSONFunc,
	}
}

type recordsStorageValue struct {
	baseStateValue
	record     istructs.IRecord
	toJSONFunc toJSONFunc
}

func (v *recordsStorageValue) AsInt32(name string) int32        { return v.record.AsInt32(name) }
func (v *recordsStorageValue) AsInt64(name string) int64        { return v.record.AsInt64(name) }
func (v *recordsStorageValue) AsFloat32(name string) float32    { return v.record.AsFloat32(name) }
func (v *recordsStorageValue) AsFloat64(name string) float64    { return v.record.AsFloat64(name) }
func (v *recordsStorageValue) AsBytes(name string) []byte       { return v.record.AsBytes(name) }
func (v *recordsStorageValue) AsString(name string) string      { return v.record.AsString(name) }
func (v *recordsStorageValue) AsQName(name string) appdef.QName { return v.record.AsQName(name) }
func (v *recordsStorageValue) AsBool(name string) bool          { return v.record.AsBool(name) }
func (v *recordsStorageValue) AsRecordID(name string) istructs.RecordID {
	return v.record.AsRecordID(name)
}
func (v *recordsStorageValue) AsRecord(string) (record istructs.IRecord) { return v.record }
func (v *recordsStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type pLogStorageValue struct {
	baseStateValue
	event      istructs.IPLogEvent
	offset     int64
	toJSONFunc toJSONFunc
}

func (v *pLogStorageValue) AsInt64(name string) int64 {
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
	return 0
}
func (v *pLogStorageValue) AsBool(string) bool { return v.event.Synced() }
func (v *pLogStorageValue) AsRecord(string) istructs.IRecord {
	return v.event.ArgumentObject().AsRecord()
}
func (v *pLogStorageValue) AsEvent(string) istructs.IDbEvent { return v.event }
func (v *pLogStorageValue) AsValue(name string) istructs.IStateValue {
	if name != Field_CUDs {
		panic(ErrNotSupported)
	}
	sv := &cudsStorageValue{}
	_ = v.event.CUDs(func(rec istructs.ICUDRow) error {
		sv.cuds = append(sv.cuds, rec)
		return nil
	})
	return sv
}
func (v *pLogStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type wLogStorageValue struct {
	baseStateValue
	event      istructs.IWLogEvent
	offset     int64
	toJSONFunc toJSONFunc
}

func (v *wLogStorageValue) AsInt64(name string) int64 {
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
func (v *wLogStorageValue) AsBool(_ string) bool          { return v.event.Synced() }
func (v *wLogStorageValue) AsQName(_ string) appdef.QName { return v.event.QName() }
func (v *wLogStorageValue) AsEvent(_ string) (event istructs.IDbEvent) {
	return v.event
}
func (v *wLogStorageValue) AsRecord(_ string) (record istructs.IRecord) {
	return v.event.ArgumentObject().AsRecord()
}
func (v *wLogStorageValue) AsValue(name string) istructs.IStateValue {
	if name != Field_CUDs {
		panic(ErrNotSupported)
	}
	sv := &cudsStorageValue{}
	_ = v.event.CUDs(func(rec istructs.ICUDRow) error {
		sv.cuds = append(sv.cuds, rec)
		return nil
	})
	return sv
}
func (v *wLogStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type sendMailStorageKeyBuilder struct {
	*keyBuilder
	to  []string
	cc  []string
	bcc []string
}

func (b *sendMailStorageKeyBuilder) PutString(name string, value string) {
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

type httpStorageKeyBuilder struct {
	*keyBuilder
	headers map[string]string
}

func newHTTPStorageKeyBuilder() *httpStorageKeyBuilder {
	return &httpStorageKeyBuilder{
		keyBuilder: newKeyBuilder(HTTPStorage, appdef.NullQName),
		headers:    make(map[string]string),
	}
}

func (b *httpStorageKeyBuilder) PutString(name string, value string) {
	switch name {
	case Field_Header:
		trim := func(v string) string { return strings.Trim(v, " \n\r\t") }
		ss := strings.SplitN(value, ":", 2)
		b.headers[trim(ss[0])] = trim(ss[1])
	default:
		b.keyBuilder.PutString(name, value)
	}
}

func (b *httpStorageKeyBuilder) method() string {
	if v, ok := b.keyBuilder.data[Field_Method]; ok {
		return v.(string)
	}
	return http.MethodGet
}
func (b *httpStorageKeyBuilder) url() string {
	if v, ok := b.keyBuilder.data[Field_Url]; ok {
		return v.(string)
	}
	panic(fmt.Errorf("'url': %w", ErrNotFound))
}
func (b *httpStorageKeyBuilder) body() io.Reader {
	if v, ok := b.keyBuilder.data[Field_Body]; ok {
		return bytes.NewReader(v.([]byte))
	}
	return nil
}
func (b *httpStorageKeyBuilder) timeout() time.Duration {
	if v, ok := b.keyBuilder.data[Field_HTTPClientTimeoutMilliseconds]; ok {
		return time.Duration(v.(int64)) * time.Millisecond
	}
	return defaultHTTPClientTimeout
}
func (b *httpStorageKeyBuilder) String() string {
	ss := make([]string, 0, httpStorageKeyBuilderStringerSliceCap)
	ss = append(ss, b.method())
	ss = append(ss, b.url())
	if v, ok := b.keyBuilder.data[Field_Body]; ok {
		ss = append(ss, string(v.([]byte)))
	}
	return strings.Join(ss, " ")
}

type httpStorageValue struct {
	istructs.IStateValue
	body       []byte
	header     map[string][]string
	statusCode int
	toJSONFunc toJSONFunc
}

func (v *httpStorageValue) AsBytes(string) []byte { return v.body }
func (v *httpStorageValue) AsInt32(string) int32  { return int32(v.statusCode) }
func (v *httpStorageValue) AsString(name string) string {
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
func (v *httpStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type appSecretsStorageValue struct {
	baseStateValue
	content    string
	toJSONFunc toJSONFunc
}

func (v *appSecretsStorageValue) AsString(string) string { return v.content }
func (v *appSecretsStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

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

type subjectStorageValue struct {
	baseStateValue
	kind        int32
	profileWSID int64
	name        string
	token       string
	toJSONFunc  toJSONFunc
}

func (v *subjectStorageValue) AsInt64(name string) int64 {
	switch name {
	case Field_ProfileWSID:
		return v.profileWSID
	default:
		return 0
	}
}
func (v *subjectStorageValue) AsInt32(name string) int32 {
	switch name {
	case Field_Kind:
		return v.kind
	default:
		return 0
	}
}
func (v *subjectStorageValue) AsString(name string) string {
	switch name {
	case Field_Name:
		return v.name
	case Field_Token:
		return v.token
	default:
		return ""
	}
}
func (v *subjectStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type viewRecordsStorageValue struct {
	baseStateValue
	value      istructs.IValue
	toJSONFunc toJSONFunc
}

func (v *viewRecordsStorageValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *viewRecordsStorageValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *viewRecordsStorageValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *viewRecordsStorageValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *viewRecordsStorageValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *viewRecordsStorageValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *viewRecordsStorageValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *viewRecordsStorageValue) AsBool(name string) bool          { return v.value.AsBool(name) }
func (v *viewRecordsStorageValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}
func (v *viewRecordsStorageValue) AsRecord(name string) istructs.IRecord {
	return v.value.AsRecord(name)
}
func (v *viewRecordsStorageValue) ToJSON(opts ...interface{}) (string, error) {
	return v.toJSONFunc(v, opts...)
}

type cudsStorageValue struct {
	istructs.IStateValue
	cuds []istructs.ICUDRow
}

func (v *cudsStorageValue) Length() int { return len(v.cuds) }
func (v *cudsStorageValue) GetAsValue(index int) istructs.IStateValue {
	return &cudRowStorageValue{value: v.cuds[index]}
}

type cudRowStorageValue struct {
	baseStateValue
	value istructs.ICUDRow
}

func (v *cudRowStorageValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *cudRowStorageValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *cudRowStorageValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *cudRowStorageValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *cudRowStorageValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *cudRowStorageValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *cudRowStorageValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *cudRowStorageValue) AsBool(name string) bool {
	if name == Field_IsNew {
		return v.value.IsNew()
	}
	return v.value.AsBool(name)
}
func (v *cudRowStorageValue) AsRecordID(name string) istructs.RecordID {
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
func (v *baseStateValue) ToJSON(...interface{}) (string, error)           { panic(errNotImplemented) }
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

type cmdResultKeyBuilder struct {
	*keyBuilder
}

func newCmdResultKeyBuilder(entity appdef.QName) *cmdResultKeyBuilder {
	return nil // type of this "nil" will be *cmdResultKeyBuilder -> will be fired at state.getStorageID()
}

type cmdResultValueBuilder struct {
	istructs.IStateValueBuilder
	cmdResultBuilder istructs.IObjectBuilder
}

func (c *cmdResultValueBuilder) PutInt32(name string, value int32) {
	c.cmdResultBuilder.PutInt32(name, value)
}

func (c *cmdResultValueBuilder) PutInt64(name string, value int64) {
	c.cmdResultBuilder.PutInt64(name, value)
}
func (c *cmdResultValueBuilder) PutBytes(name string, value []byte) {
	c.cmdResultBuilder.PutBytes(name, value)
}
func (c *cmdResultValueBuilder) PutString(name, value string) {
	c.cmdResultBuilder.PutString(name, value)
}
func (c *cmdResultValueBuilder) PutBool(name string, value bool) {
	c.cmdResultBuilder.PutBool(name, value)
}
func (c *cmdResultValueBuilder) PutChars(name string, value string) {
	c.cmdResultBuilder.PutChars(name, value)
}
func (c *cmdResultValueBuilder) PutFloat32(name string, value float32) {
	c.cmdResultBuilder.PutFloat32(name, value)
}
func (c *cmdResultValueBuilder) PutFloat64(name string, value float64) {
	c.cmdResultBuilder.PutFloat64(name, value)
}
func (c *cmdResultValueBuilder) PutQName(name string, value appdef.QName) {
	c.cmdResultBuilder.PutQName(name, value)
}
func (c *cmdResultValueBuilder) PutNumber(name string, value float64) {
	c.cmdResultBuilder.PutNumber(name, value)
}
func (c *cmdResultValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	c.cmdResultBuilder.PutRecordID(name, value)
}
