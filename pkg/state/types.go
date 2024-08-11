/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"encoding/base64"
	"io"
	"maps"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state/smtptest"

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

type FederationCommandHandler = func(owner, appname string, wsid istructs.WSID, command appdef.QName, body string) (statusCode int, newIDs map[string]int64, result string, err error)
type FederationBlobHandler = func(owner, appname string, wsid istructs.WSID, blobId int64) (result []byte, err error)
type UniquesHandler = func(entity appdef.QName, wsid istructs.WSID, data map[string]interface{}) (istructs.RecordID, error)

type eventsFunc func() istructs.IEvents
type recordsFunc func() istructs.IRecords

type StateOptFunc func(opts *StateOpts)

type IHttpClient interface {
	Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error)
}

type StateOpts struct {
	messages                 chan smtptest.Message
	federationCommandHandler FederationCommandHandler
	federationBlobHandler    FederationBlobHandler
	customHttpClient         IHttpClient
	uniquesHandler           UniquesHandler
}

func WithEmailMessagesChan(messages chan smtptest.Message) StateOptFunc {
	return func(opts *StateOpts) {
		opts.messages = messages
	}
}

func WithCustomHttpClient(client IHttpClient) StateOptFunc {
	return func(opts *StateOpts) {
		opts.customHttpClient = client
	}
}

func WithFedearationCommandHandler(handler FederationCommandHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.federationCommandHandler = handler
	}
}

func WithFederationBlobHandler(handler FederationBlobHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.federationBlobHandler = handler
	}
}

func WithUniquesHandler(handler UniquesHandler) StateOptFunc {
	return func(opts *StateOpts) {
		opts.uniquesHandler = handler
	}
}

type ApplyBatchItem struct {
	Key   istructs.IStateKeyBuilder
	Value istructs.IStateValueBuilder
	IsNew bool
}

type GetBatchItem struct {
	Key   istructs.IStateKeyBuilder
	Value istructs.IStateValue
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

type key struct {
	istructs.IKey
	data map[string]interface{}
}

func (k *key) AsInt64(name string) int64 { return k.data[name].(int64) }

type wsTypeKey struct {
	wsid     istructs.WSID
	appQName appdef.AppQName
}
