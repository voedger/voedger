/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"container/list"
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/utils/federation"
)

type PartitionIDFunc func() istructs.PartitionID
type WSIDFunc func() istructs.WSID
type N10nFunc func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset)
type AppStructsFunc func() istructs.IAppStructs
type CUDFunc func() istructs.ICUD
type ObjectBuilderFunc func() istructs.IObjectBuilder
type PrincipalsFunc func() []iauthnz.Principal
type TokenFunc func() string
type PLogEventFunc func() istructs.IPLogEvent
type CommandPrepareArgsFunc func() istructs.CommandPrepareArgs
type ArgFunc func() istructs.IObject
type UnloggedArgFunc func() istructs.IObject
type WLogOffsetFunc func() istructs.Offset
type FederationFunc func() federation.IFederation
type QNameFunc func() appdef.QName
type TokensFunc func() itokens.ITokens
type PrepareArgsFunc func() istructs.PrepareArgs
type ExecQueryCallbackFunc func() istructs.ExecQueryCallback
type CommandProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, intentsLimit int, cmdResultBuilderFunc ObjectBuilderFunc, execCmdArgsFunc CommandPrepareArgsFunc, argFunc ArgFunc, unloggedArgFunc UnloggedArgFunc, wlogOffsetFunc WLogOffsetFunc, opts ...StateOptFunc) IHostState
type SyncActualizerStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit int, opts ...StateOptFunc) IHostState
type QueryProcessorStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, principalPayloadFunc PrincipalsFunc, tokenFunc TokenFunc, itokens itokens.ITokens, execQueryArgsFunc PrepareArgsFunc, argFunc ArgFunc, resultBuilderFunc ObjectBuilderFunc, federation federation.IFederation, queryCallbackFunc ExecQueryCallbackFunc, opts ...StateOptFunc) IHostState
type AsyncActualizerStateFactory func(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, tokensFunc itokens.ITokens, federationFunc federation.IFederation, intentsLimit, bundlesLimit int, opts ...StateOptFunc) IBundledHostState

type eventsFunc func() istructs.IEvents
type recordsFunc func() istructs.IRecords

type ApplyBatchItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
	isNew bool
}

type GetBatchItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValue
}

type StateOptFunc func(opts *stateOpts)

func WithEmailMessagesChan(messages chan smtptest.Message) StateOptFunc {
	return func(opts *stateOpts) {
		opts.messages = messages
	}
}

func WithCustomHttpClient(client IHttpClient) StateOptFunc {
	return func(opts *stateOpts) {
		opts.customHttpClient = client
	}
}

func WithFedearationCommandHandler(handler FederationCommandHandler) StateOptFunc {
	return func(opts *stateOpts) {
		opts.federationCommandHandler = handler
	}
}

func WithFederationBlobHandler(handler FederationBlobHandler) StateOptFunc {
	return func(opts *stateOpts) {
		opts.federationBlobHandler = handler
	}
}

func WithUniquesHandler(handler UniquesHandler) StateOptFunc {
	return func(opts *stateOpts) {
		opts.uniquesHandler = handler
	}
}

type mapKeyBuilder struct {
	data    map[string]interface{}
	storage appdef.QName
	entity  appdef.QName
}

func newMapKeyBuilder(storage, entity appdef.QName) *mapKeyBuilder {
	return &mapKeyBuilder{
		data:    make(map[string]interface{}),
		storage: storage,
		entity:  entity,
	}
}

func (b *mapKeyBuilder) Storage() appdef.QName                            { return b.storage }
func (b *mapKeyBuilder) Entity() appdef.QName                             { return b.entity }
func (b *mapKeyBuilder) PutInt32(name string, value int32)                { b.data[name] = value }
func (b *mapKeyBuilder) PutInt64(name string, value int64)                { b.data[name] = value }
func (b *mapKeyBuilder) PutFloat32(name string, value float32)            { b.data[name] = value }
func (b *mapKeyBuilder) PutFloat64(name string, value float64)            { b.data[name] = value }
func (b *mapKeyBuilder) PutBytes(name string, value []byte)               { b.data[name] = value }
func (b *mapKeyBuilder) PutString(name string, value string)              { b.data[name] = value }
func (b *mapKeyBuilder) PutQName(name string, value appdef.QName)         { b.data[name] = value }
func (b *mapKeyBuilder) PutBool(name string, value bool)                  { b.data[name] = value }
func (b *mapKeyBuilder) PutRecordID(name string, value istructs.RecordID) { b.data[name] = value }
func (b *mapKeyBuilder) PutNumber(string, float64)                        { panic(ErrNotSupported) }
func (b *mapKeyBuilder) PutChars(string, string)                          { panic(ErrNotSupported) }
func (b *mapKeyBuilder) PutFromJSON(j map[string]any)                     { maps.Copy(b.data, j) }
func (b *mapKeyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) }
func (b *mapKeyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) }
func (b *mapKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*mapKeyBuilder)
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
func (b *mapKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error) { panic(ErrNotSupported) }

type viewKeyBuilder struct {
	istructs.IKeyBuilder
	wsid istructs.WSID
	view appdef.QName
}

func (b *viewKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_View_Field_WSID {
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
	return sys.Storage_View
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

// used in tests
func (b *viewValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	bThis, err := b.IValueBuilder.ToBytes()
	if err != nil {
		panic(err)
	}

	bSrc, err := src.ToBytes()
	if err != nil {
		panic(err)
	}

	return reflect.DeepEqual(bThis, bSrc)
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

type jsonArrayValue struct {
	baseStateValue
	array []interface{}
}

func (v *jsonArrayValue) GetAsString(i int) string      { return v.array[i].(string) }
func (v *jsonArrayValue) GetAsBytes(i int) []byte       { return v.array[i].([]byte) }
func (v *jsonArrayValue) GetAsInt32(i int) int32        { return v.array[i].(int32) }
func (v *jsonArrayValue) GetAsInt64(i int) int64        { return v.array[i].(int64) }
func (v *jsonArrayValue) GetAsFloat32(i int) float32    { return v.array[i].(float32) }
func (v *jsonArrayValue) GetAsFloat64(i int) float64    { return v.array[i].(float64) }
func (v *jsonArrayValue) GetAsQName(i int) appdef.QName { return v.array[i].(appdef.QName) }
func (v *jsonArrayValue) GetAsBool(i int) bool          { return v.array[i].(bool) }
func (v *jsonArrayValue) GetAsValue(i int) (result istructs.IStateValue) {
	switch v := v.array[i].(type) {
	case map[string]interface{}:
		return &jsonValue{json: v}
	case []interface{}:
		return &jsonArrayValue{array: v}
	default:
		panic(errUnexpectedType(v))
	}
}
func (v *jsonArrayValue) Length() int {
	return len(v.array)
}

type jsonValue struct {
	baseStateValue
	json map[string]interface{}
}

func (v *jsonValue) AsInt32(name string) int32 {
	if v, ok := v.json[name]; ok {
		return int32(v.(float64))
	}
	panic(errInt32FieldUndefined(name))
}
func (v *jsonValue) AsInt64(name string) int64 {
	if v, ok := v.json[name]; ok {
		return v.(int64)
	}
	panic(errInt64FieldUndefined(name))
}
func (v *jsonValue) AsFloat32(name string) float32 {
	if v, ok := v.json[name]; ok {
		return v.(float32)
	}
	panic(errFloat32FieldUndefined(name))
}
func (v *jsonValue) AsFloat64(name string) float64 {
	if v, ok := v.json[name]; ok {
		return v.(float64)
	}
	panic(errFloat64FieldUndefined(name))
}
func (v *jsonValue) AsBytes(name string) []byte {
	if v, ok := v.json[name]; ok {
		data, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			panic(err)
		}
		return data
	}
	panic(errBytesFieldUndefined(name))
}
func (v *jsonValue) AsString(name string) string {
	if v, ok := v.json[name]; ok {
		return v.(string)
	}
	panic(errStringFieldUndefined(name))
}
func (v *jsonValue) AsQName(name string) appdef.QName {
	if v, ok := v.json[name]; ok {
		return appdef.MustParseQName(v.(string))
	}
	panic(errQNameFieldUndefined(name))
}
func (v *jsonValue) AsBool(name string) bool {
	if v, ok := v.json[name]; ok {
		return v.(string) == "true"
	}
	panic(errBoolFieldUndefined(name))
}
func (v *jsonValue) AsRecordID(name string) istructs.RecordID {
	if v, ok := v.json[name]; ok {
		return istructs.RecordID(v.(float64))
	}
	panic(errRecordIDFieldUndefined(name))
}
func (v *jsonValue) RecordIDs(includeNulls bool, cb func(string, istructs.RecordID)) {}
func (v *jsonValue) FieldNames(cb func(string)) {
	for name := range v.json {
		cb(name)
	}
}
func (v *jsonValue) AsValue(name string) (result istructs.IStateValue) {
	if v, ok := v.json[name]; ok {
		switch v := v.(type) {
		case map[string]interface{}:
			return &jsonValue{json: v}
		case []interface{}:
			return &jsonArrayValue{array: v}
		default:
			panic(errUnexpectedType(v))
		}
	}
	panic(errValueFieldUndefined(name))
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
		panic(errValueFieldUndefined(name))
	}
	return
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
	if name == sys.Storage_Http_Field_Header {
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
	if name == sys.Storage_Event_Field_ErrStr {
		return v.error.ErrStr()
	}
	panic(ErrNotSupported)
}

func (v *eventErrorValue) AsBool(name string) bool {
	if name == sys.Storage_Event_Field_ValidEvent {
		return v.error.ValidEvent()
	}
	panic(ErrNotSupported)
}

func (v *eventErrorValue) AsQName(name string) appdef.QName {
	if name == sys.Storage_Event_Field_QNameFromParams {
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
	if name == sys.CUDs_Field_IsNew {
		return v.value.IsNew()
	}
	return v.value.AsBool(name)
}
func (v *cudRowValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}

type baseStateValue struct{}

func (v *baseStateValue) AsInt32(name string) int32        { panic(errStringFieldUndefined(name)) }
func (v *baseStateValue) AsInt64(name string) int64        { panic(errInt64FieldUndefined(name)) }
func (v *baseStateValue) AsFloat32(name string) float32    { panic(errFloat32FieldUndefined(name)) }
func (v *baseStateValue) AsFloat64(name string) float64    { panic(errFloat64FieldUndefined(name)) }
func (v *baseStateValue) AsBytes(name string) []byte       { panic(errBytesFieldUndefined(name)) }
func (v *baseStateValue) AsString(name string) string      { panic(errStringFieldUndefined(name)) }
func (v *baseStateValue) AsQName(name string) appdef.QName { panic(errQNameFieldUndefined(name)) }
func (v *baseStateValue) AsBool(name string) bool          { panic(errBoolFieldUndefined(name)) }
func (v *baseStateValue) AsValue(name string) istructs.IStateValue {
	panic(errValueFieldUndefined(name))
}
func (v *baseStateValue) AsRecordID(name string) istructs.RecordID {
	panic(errRecordIDFieldUndefined(name))
}
func (v *baseStateValue) AsRecord(name string) istructs.IRecord           { panic(errNotImplemented) }
func (v *baseStateValue) AsEvent(name string) istructs.IDbEvent           { panic(errNotImplemented) }
func (v *baseStateValue) RecordIDs(bool, func(string, istructs.RecordID)) { panic(errNotImplemented) }
func (v *baseStateValue) FieldNames(func(string))                         { panic(errNotImplemented) }
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

type wsTypeKey struct {
	wsid     istructs.WSID
	appQName appdef.AppQName
}

type wsTypeVailidator struct {
	appStructsFunc AppStructsFunc
	wsidKinds      map[wsTypeKey]appdef.QName
}

func newWsTypeValidator(appStructsFunc AppStructsFunc) wsTypeVailidator {
	return wsTypeVailidator{
		appStructsFunc: appStructsFunc,
		wsidKinds:      make(map[wsTypeKey]appdef.QName),
	}
}

// Returns NullQName if not found
func (v *wsTypeVailidator) getWSIDKind(wsid istructs.WSID, entity appdef.QName) (appdef.QName, error) {
	key := wsTypeKey{wsid: wsid, appQName: v.appStructsFunc().AppQName()}
	wsKind, ok := v.wsidKinds[key]
	if !ok {
		wsDesc, err := v.appStructsFunc().Records().GetSingleton(wsid, qNameCDocWorkspaceDescriptor)
		if err != nil {
			// notest
			return appdef.NullQName, err
		}
		if wsDesc.QName() == appdef.NullQName {
			if v.appStructsFunc().AppDef().WorkspaceByDescriptor(entity) != nil {
				// Special case. sys.CreateWorkspace creates WSKind while WorkspaceDescriptor is not applied yet.
				return entity, nil
			}
			return appdef.NullQName, fmt.Errorf("%w: %d", errWorkspaceDescriptorNotFound, wsid)
		}
		wsKind = wsDesc.AsQName(field_WSKind)
		if len(v.wsidKinds) < wsidTypeValidatorCacheSize {
			v.wsidKinds[key] = wsKind
		}
	}
	return wsKind, nil
}

func (v *wsTypeVailidator) validate(wsid istructs.WSID, entity appdef.QName) error {
	if entity == qNameCDocWorkspaceDescriptor {
		return nil // This QName always can be read and write. Otherwise sys.CreateWorkspace is not able to create descriptor.
	}
	if wsid != istructs.NullWSID && v.appStructsFunc().Records() != nil { // NullWSID only stores actualizer offsets
		wsKind, err := v.getWSIDKind(wsid, entity)
		if err != nil {
			// notest
			return err
		}
		ws := v.appStructsFunc().AppDef().WorkspaceByDescriptor(wsKind)
		if ws == nil {
			// notest
			return errDescriptorForUndefinedWorkspace
		}
		if ws.TypeByName(entity) == nil {
			return typeIsNotDefinedInWorkspaceWithDescriptor(entity, wsKind)
		}
	}
	return nil
}

type baseKeyBuilder struct {
	istructs.IStateKeyBuilder
	entity appdef.QName
}

func (b *baseKeyBuilder) Storage() appdef.QName {
	panic(errNotImplemented)
}
func (b *baseKeyBuilder) Entity() appdef.QName {
	return b.entity
}
func (b *baseKeyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) ToBytes(istructs.WSID) (pk, cc []byte, err error) { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) PutInt32(name appdef.FieldName, value int32) {
	panic(errInt32FieldUndefined(name))
}
func (b *baseKeyBuilder) PutInt64(name appdef.FieldName, value int64) {
	panic(errInt64FieldUndefined(name))
}
func (b *baseKeyBuilder) PutFloat32(name appdef.FieldName, value float32) {
	panic(errFloat32FieldUndefined(name))
}
func (b *baseKeyBuilder) PutFloat64(name appdef.FieldName, value float64) {
	panic(errFloat64FieldUndefined(name))
}

// Puts value into bytes or raw data field.
func (b *baseKeyBuilder) PutBytes(name appdef.FieldName, value []byte) {
	panic(errBytesFieldUndefined(name))
}

// Puts value into string or raw data field.
func (b *baseKeyBuilder) PutString(name appdef.FieldName, value string) {
	panic(errStringFieldUndefined(name))
}

func (b *baseKeyBuilder) PutQName(name appdef.FieldName, value appdef.QName) {
	panic(errQNameFieldUndefined(name))
}
func (b *baseKeyBuilder) PutBool(name appdef.FieldName, value bool) {
	panic(errBoolFieldUndefined(name))
}
func (b *baseKeyBuilder) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	panic(errRecordIDFieldUndefined(name))
}
func (b *baseKeyBuilder) PutNumber(name appdef.FieldName, value float64) {
	panic(errNumberFieldUndefined(name))
}
func (b *baseKeyBuilder) PutChars(name appdef.FieldName, value string) {
	panic(errCharsFieldUndefined(name))
}
func (b *baseKeyBuilder) PutFromJSON(map[appdef.FieldName]any) {
	panic(ErrNotSupported)
}
func (b *baseKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	panic(errNotImplemented)
}

type baseValueBuilder struct {
	istructs.IStateValueBuilder
}

func (b *baseValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	return false
}
func (b *baseValueBuilder) PutInt32(name string, value int32) {
	panic(errInt32FieldUndefined(name))
}
func (b *baseValueBuilder) PutInt64(name string, value int64) {
	panic(errInt64FieldUndefined(name))
}
func (b *baseValueBuilder) PutBytes(name string, value []byte) {
	panic(errBytesFieldUndefined(name))
}
func (b *baseValueBuilder) PutString(name, value string)       { panic(errStringFieldUndefined(name)) }
func (b *baseValueBuilder) PutBool(name string, value bool)    { panic(errBoolFieldUndefined(name)) }
func (b *baseValueBuilder) PutChars(name string, value string) { panic(errCharsFieldUndefined(name)) }
func (b *baseValueBuilder) PutFloat32(name string, value float32) {
	panic(errFloat32FieldUndefined(name))
}
func (b *baseValueBuilder) PutFloat64(name string, value float64) {
	panic(errFloat64FieldUndefined(name))
}
func (b *baseValueBuilder) PutQName(name string, value appdef.QName) {
	panic(errQNameFieldUndefined(name))
}
func (b *baseValueBuilder) PutNumber(name string, value float64) {
	panic(errNumberFieldUndefined(name))
}
func (b *baseValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	panic(errRecordIDFieldUndefined(name))
}
func (b *baseValueBuilder) BuildValue() istructs.IStateValue {
	panic(errNotImplemented)
}
