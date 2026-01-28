/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package teststate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type IODOc interface {
	IAmODoc()
}

// TODO: eliminate usage of IAppStructs here

// generalTestState is a test state
type generalTestState struct {
	testState

	extensionFunc func()
	funcRunner    *sync.Once

	// recordItems is to store records
	recordItems []recordItem
	// requiredRecordItems is to store required items
	requiredRecordItems []recordItem
	// cudRows is to store cud rows
	cudRows []recordItem
	// view records
	viewRecords []recordItem
}

func (gts *generalTestState) getMockedStorage(storageQName appdef.QName) (*storages.MockedStorage, bool) {
	mockedState, ok1 := gts.IState.(*stateprovide.MockedState)
	if !ok1 {
		panic("failed to get mocked state")
	}

	mockedStorage, ok2 := mockedState.GetMockedStorage(storageQName)
	if !ok2 {
		panic(fmt.Sprintf("failed to get mocked storage for %s", storageQName.String()))
	}

	return mockedStorage, true
}

func (gts *generalTestState) intentSingletonInsert(fQName IFullQName, fileRef string, keyValueList ...any) {
	gts.addRequiredItems(fQName, 0, true, true, false, fileRef, keyValueList...)
}

func (gts *generalTestState) intentSingletonUpdate(fQName IFullQName, fileRef string, keyValueList ...any) {
	gts.addRequiredItems(fQName, 0, true, false, false, fileRef, keyValueList...)
}

func (gts *generalTestState) intentRecordInsert(fQName IFullQName, id istructs.RecordID, fileRef string, keyValueList ...any) {
	gts.addRequiredItems(fQName, id, false, true, false, fileRef, keyValueList...)
}

func (gts *generalTestState) intentRecordUpdate(fQName IFullQName, id istructs.RecordID, fileRef string, keyValueList ...any) {
	gts.addRequiredItems(fQName, id, false, false, false, fileRef, keyValueList...)
}

func (gts *generalTestState) stateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) {
	isSingleton := gts.isSingletone(fQName)
	if isSingleton {
		panic("use SingletonRecord method for singletons")
	}

	gts.record(fQName, id, isSingleton, keyValueList...)
}

// SingletonRecord adds a singleton record to the state
// Implemented in own method because of ID for singletons are generated under-the-hood and
// we can not insert singletons with our own IDs
func (gts *generalTestState) stateSingletonRecord(fQName IFullQName, keyValueList ...any) {
	isSingleton := gts.isSingletone(fQName)
	if !isSingleton {
		panic("use Record method for non-singleton entities")
	}

	gts.record(fQName, istructs.MinReservedRecordID, isSingleton, keyValueList...)
}

func (gts *generalTestState) getQNameFromFQName(fQName IFullQName) appdef.QName {
	localPkgName := gts.appDef.PackageLocalName(fQName.PkgPath())
	return appdef.NewQName(localPkgName, fQName.Entity())
}

func (gts *generalTestState) isSingletone(fQName IFullQName) bool {
	qName := gts.getQNameFromFQName(fQName)

	iSingleton := appdef.Singleton(gts.appDef.Type, qName)
	return iSingleton != nil && iSingleton.Singleton()
}

func (gts *generalTestState) isView(fQName IFullQName) bool {
	qName := gts.getQNameFromFQName(fQName)

	iView := appdef.View(gts.appDef.Type, qName)
	return iView != nil
}

func (gts *generalTestState) runExtensionFunc() {
	if gts.extensionFunc != nil {
		gts.funcRunner.Do(gts.extensionFunc)
	}
}

func (gts *generalTestState) putCudRows() {
	if len(gts.cudRows) == 0 {
		return
	}

	mockedEventStorage, ok := gts.getMockedStorage(sys.Storage_Event)
	if !ok {
		panic("failed to get mocked event storage")
	}

	for _, item := range gts.cudRows {
		kvMap, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, msgFailedToParseKeyValues)

		localQName := gts.getQNameFromFQName(item.entity)

		mockedEventObject := &coreutils.TestObject{
			Name:        localQName,
			Data:        kvMap,
			Containers_: make(map[string][]*coreutils.TestObject),
		}
		mockedEventObject.Data[appdef.SystemField_ID] = item.id
		mockedEventObject.Data[sys.Storage_Event_Field_Workspace] = gts.commandWSID
		mockedEventObject.Data[sys.Storage_Event_Field_QName] = localQName

		// writing an event object directly to the storage
		mockedEventStorage.PutRecord(uint64(item.id), mockedEventObject)
	}
}

func (gts *generalTestState) addRequiredItems(
	fQName IFullQName,
	id istructs.RecordID,
	isSingleton, isNew, isView bool,
	fileRef string,
	keyValueList ...any,
) {
	gts.requiredRecordItems = append(gts.requiredRecordItems, recordItem{
		entity:        fQName,
		qName:         gts.getQNameFromFQName(fQName),
		id:            id,
		isSingleton:   isSingleton,
		isNew:         isNew,
		isView:        isView,
		fileReference: fileRef,
		keyValueList:  keyValueList,
	})
}

// recoverPanicInTestState must be called in defer to recover panic in the test state
func (gts *generalTestState) recoverPanicInTestState() {
	r := recover()
	recoveredError, ok := r.(error)
	if ok {
		require.Fail(gts.t, recoveredError.Error())
	}
}

// requireIntent checks if the intent exists in the state
// Parameters:
// fQName - full qname of the entity
// id - record id (unused for singletons)
// isSingletone - if the entity is a singleton
// isInsertIntent - if the intent is insert or update
// isView - if the entity is a view
// keyValueList - list of key-value pairs
func (gts *generalTestState) requireIntent(r recordItem) {
	kb := gts.keyBuilder(r)

	vb, isNew := gts.IState.FindIntentWithOpKind(kb)
	if vb == nil {
		require.Fail(gts.t, fmt.Sprintf("intent not found: %s", r.fileReference))
		return
	}

	m, err := parseKeyValues(r.keyValueList)
	require.NoError(gts.t, err, msgFailedToParseKeyValues)

	_, mapOfValues := splitKeysFromValues(r.entity, m)

	localQName := gts.getQNameFromFQName(r.entity)
	require.Equalf(gts.t, r.isNew, isNew, "%s: intent kind mismatch", localQName.String())
	gts.equalValues(vb, mapOfValues, r.fileReference)
}

func (gts *generalTestState) equalValues(vb istructs.IStateValueBuilder, expectedValues map[string]any, fileReference string) {
	var fileRefSuffix string
	if len(fileReference) != 0 {
		fileRefSuffix = fmt.Sprintf(": %s", fileReference)
	}

	if vb == nil {
		require.Fail(gts.t, "expected value builder is nil")
		return
	}
	value := vb.BuildValue()
	if value == nil {
		require.Fail(gts.t, "value builder does not support equalValues operation")
		return
	}

	notEqualMsg := fmt.Sprintf("values are not equal%s", fileRefSuffix)
	for expectedKey, expectedValue := range expectedValues {
		switch t := expectedValue.(type) {
		case int8:
			require.Equal(gts.t, int32(t), value.AsInt32(expectedKey), notEqualMsg)
		case int16:
			require.Equal(gts.t, int32(t), value.AsInt32(expectedKey), notEqualMsg)
		case int32:
			require.Equal(gts.t, t, value.AsInt32(expectedKey), notEqualMsg)
		case int64:
			require.Equal(gts.t, t, value.AsInt64(expectedKey), notEqualMsg)
		case int:
			require.Equal(gts.t, int64(t), value.AsInt64(expectedKey), notEqualMsg)
		case float32:
			require.Equal(gts.t, t, value.AsFloat32(expectedKey), notEqualMsg)
		case float64:
			require.Equal(gts.t, t, value.AsFloat64(expectedKey), notEqualMsg)
		case []byte:
			require.Equal(gts.t, t, value.AsBytes(expectedKey), notEqualMsg)
		case string:
			require.Equal(gts.t, t, value.AsString(expectedKey), notEqualMsg)
		case bool:
			require.Equal(gts.t, t, value.AsBool(expectedKey), notEqualMsg)
		case appdef.QName:
			require.Equal(gts.t, t, value.AsQName(expectedKey), notEqualMsg)
		case istructs.IStateValue:
			require.Equal(gts.t, t, value.AsValue(expectedKey), notEqualMsg)
		case json.Number:
			int64Value, err := t.Int64()
			require.NoError(gts.t, err)

			require.Equal(gts.t, int64Value, value.AsInt64(expectedKey), notEqualMsg)
		default:
			require.Fail(gts.t, "unsupported value type")
		}
	}
}

func (gts *generalTestState) keyBuilder(r recordItem) istructs.IStateKeyBuilder {
	var err error
	var kb istructs.IStateKeyBuilder

	localQName := gts.getQNameFromFQName(r.entity)
	if r.isView {
		kb, err = gts.IState.KeyBuilder(sys.Storage_View, localQName)
	} else {
		kb, err = gts.IState.KeyBuilder(sys.Storage_Record, localQName)
	}

	require.NoError(gts.t, err, "IState.KeyBuilder: failed to create key builder")

	switch {
	case r.isSingleton:
		kb.PutBool(sys.Storage_Record_Field_IsSingleton, true)
	case !r.isView:
		kb.PutRecordID(sys.Storage_Record_Field_ID, r.id)
	case r.isView:
		m, err := parseKeyValues(r.keyValueList)
		require.NoError(gts.t, err, msgFailedToParseKeyValues)

		mapOfKeys, _ := splitKeysFromValues(r.entity, m)

		kb.PutFromJSON(mapOfKeys)
	}

	return kb
}

func (gts *generalTestState) record(fQName IFullQName, id istructs.RecordID, isSingleton bool, keyValueList ...any) {
	qName := gts.getQNameFromFQName(fQName)

	// check if the record already exists
	_ = slices.ContainsFunc(gts.recordItems, func(i recordItem) bool {
		if i.entity == fQName {
			if isSingleton {
				panic(fmt.Errorf("singletone %s already exists", qName.String()))
			}
			if i.id == id {
				panic(fmt.Errorf("record with entity %s and id %d already exists", qName.String(), id))
			}
		}

		return false
	})

	gts.recordItems = append(gts.recordItems, recordItem{
		entity:       fQName,
		qName:        gts.getQNameFromFQName(fQName),
		isSingleton:  isSingleton,
		id:           id,
		keyValueList: keyValueList,
	})
}

func (gts *generalTestState) putRecords() {
	mockedRecordStorage, ok := gts.getMockedStorage(sys.Storage_Record)
	if !ok {
		panic("failed to get mocked record storage")
	}

	// put records into the state
	for _, item := range gts.recordItems {
		pkgAlias := gts.appDef.PackageLocalName(item.entity.PkgPath())

		kvMap, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, msgFailedToParseKeyValues)

		kvMap[appdef.SystemField_QName] = appdef.NewQName(pkgAlias, item.entity.Entity()).String()
		kvMap[appdef.SystemField_ID] = item.id
		kvMap[sys.Storage_Record_Field_WSID] = gts.commandWSID

		mockedObject := &coreutils.TestObject{
			ID_:         item.id,
			Name:        item.qName,
			IsNew_:      item.isNew,
			Data:        kvMap,
			Containers_: make(map[string][]*coreutils.TestObject),
		}

		mockedRecordStorage.PutRecord(uint64(item.id), mockedObject)
	}

	// clear record items after they are processed
	gts.recordItems = nil
}

func (gts *generalTestState) putViewRecords() {
	mockedViewStorage, ok := gts.getMockedStorage(sys.Storage_View)
	if !ok {
		panic("failed to get mocked view storage")
	}

	for _, item := range gts.viewRecords {
		kvMap, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, msgFailedToParseKeyValues)

		kvMap[sys.Storage_Record_Field_WSID] = gts.commandWSID

		mockedObject := &coreutils.TestObject{
			ID_:         item.id,
			Name:        item.qName,
			IsNew_:      item.isNew,
			Data:        kvMap,
			Containers_: make(map[string][]*coreutils.TestObject),
		}

		kb := gts.keyBuilder(item)

		if err := mockedViewStorage.PutViewRecord(kb, mockedObject); err != nil {
			panic(fmt.Errorf("failed to put view record: %w", err))
		}
	}

	// clear view record items after they are processed
	gts.viewRecords = nil
}

func (gts *generalTestState) require() {
	// check out required allIntents
	requiredKeys := make([]istructs.IStateKeyBuilder, 0, len(gts.requiredRecordItems))
	for _, item := range gts.requiredRecordItems {
		requiredKeys = append(requiredKeys, gts.keyBuilder(item))

		gts.requireIntent(item)
	}

	// gather all intents
	allIntents := make([]intentItem, 0, gts.IState.IntentsCount())
	gts.IState.Intents(func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder, isNew bool) {
		allIntents = append(allIntents, intentItem{
			key:   key,
			value: value,
			isNew: isNew,
		})
	})

	notFoundKeys := make([]istructs.IStateKeyBuilder, 0, len(allIntents))
	// check out unexpected intents
	for _, intent := range allIntents {
		found := false
		for _, requiredKey := range requiredKeys {
			if intent.key.Equals(requiredKey) {
				found = true
				continue
			}
		}
		if !found {
			notFoundKeys = append(notFoundKeys, intent.key)
		}
	}
	require.Empty(gts.t, notFoundKeys, "unexpected intents: %v", notFoundKeys)

	// clear required record items after they are processed
	gts.requiredRecordItems = nil
}

// CommandTestState is a test state for command testing
type CommandTestState struct {
	generalTestState
}

// NewCommandTestState creates a new test state for command testing
func NewCommandTestState(t *testing.T, iCommand ICommand, extensionFunc func()) *CommandTestState {
	const wsid = istructs.WSID(1)

	cts := &CommandTestState{}

	cts.testData = make(map[string]any)
	cts.t = t
	cts.ctx = context.Background()
	cts.processorKind = ProcKind_CommandProcessor
	cts.commandWSID = wsid
	cts.secretReader = &secretReader{secrets: make(map[string][]byte)}

	// build appDef
	cts.buildAppDef()
	// build state
	cts.IState = stateprovide.ProvideMockedCommandProcessorStateFactory()(
		cts.ctx,
		IntentsLimit,
		func() istructs.IAppStructs { return cts.appStructs },
	)

	// initialize funcRunner and extensionFunc itself
	cts.funcRunner = &sync.Once{}
	cts.extensionFunc = extensionFunc

	cts.argumentObject = make(map[string]any)
	// set arguments for the command
	if len(iCommand.ArgumentEntity()) > 0 {
		cts.argumentType = appdef.NewFullQName(iCommand.ArgumentPkgPath(), iCommand.ArgumentEntity())
	}

	return cts
}

func (cts *CommandTestState) StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.stateRecord(fQName, id, keyValueList...)

	return cts
}

func (cts *CommandTestState) StateSingletonRecord(fQName IFullQName, keyValueList ...any) ICommandRunner {
	cts.stateSingletonRecord(fQName, keyValueList...)

	return cts
}

func (cts *CommandTestState) putArgument() {
	if cts.argumentObject == nil {
		return
	}

	mockedCommandContextStorage, ok := cts.getMockedStorage(sys.Storage_CommandContext)
	if !ok {
		panic("failed to get mocked command context storage")
	}

	localPkgName := cts.appDef.PackageLocalName(cts.argumentType.PkgPath())
	localQName := appdef.NewQName(localPkgName, cts.argumentType.Entity())

	mockedObject := &coreutils.TestObject{
		Containers_: make(map[string][]*coreutils.TestObject),
	}

	// setting argument object
	mockedObject.Containers_[sys.Storage_CommandContext_Field_ArgumentObject] = append(
		mockedObject.Containers_[sys.Storage_CommandContext_Field_ArgumentObject],
		&coreutils.TestObject{
			Name:        localQName,
			Data:        cts.argumentObject,
			Containers_: make(map[string][]*coreutils.TestObject),
		},
	)

	for key, value := range cts.argumentObject {
		if innerSlice, ok := value.([]any); ok {
			for _, innerValue := range innerSlice {
				mockedObject.Containers_[sys.Storage_CommandContext_Field_ArgumentObject][0].Containers_[key] = append(
					mockedObject.Containers_[sys.Storage_CommandContext_Field_ArgumentObject][0].Containers_[key],
					&coreutils.TestObject{
						Data:        innerValue.(map[string]any),
						Containers_: make(map[string][]*coreutils.TestObject),
					},
				)
			}
		}
	}
	// writing an event object directly to the storage
	mockedCommandContextStorage.PutRecord(0, mockedObject)
}

// buildAppDef alternative way of building IAppDef
func (gts *generalTestState) buildAppDef() {
	compileResult, err := compile.Compile("..")
	if err != nil {
		panic(err)
	}

	gts.appDef = compileResult.AppDef

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, compileResult.AppDefBuilder)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	for ext := range appdef.Extensions(gts.appDef.Types()) {
		if proj, ok := ext.(appdef.IProjector); ok {
			if proj.Sync() {
				cfg.AddSyncProjectors(istructs.Projector{Name: ext.QName()})
			} else {
				cfg.AddAsyncProjectors(istructs.Projector{Name: ext.QName()})
			}
		} else if cmd, ok := ext.(appdef.ICommand); ok {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmd.QName(), istructsmem.NullCommandExec))
		} else if q, ok := ext.(appdef.IQuery); ok {
			cfg.Resources.Add(istructsmem.NewCommandFunction(q.QName(), istructsmem.NullCommandExec))
		}
	}

	asf := mem.Provide(testingu.MockTime)
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider,
		isequencer.SequencesTrustLevel_0,
		nil,
	)

	structs, err := prov.BuiltIn(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}

	gts.appStructs = structs
}

func (cts *CommandTestState) ArgumentObject(id istructs.RecordID, keyValueList ...any) ICommandRunner {
	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf(fmtMsgFailedToParseKeyValues, err))
	}

	for key, value := range keyValueMap {
		v, valueExist := cts.argumentObject[key]
		if valueExist {
			panic(fmt.Errorf("key %s already exists in the argument object with value %v", key, v))
		}

		cts.argumentObject[key] = value
		if intValue, ok := value.(int); ok {
			cts.argumentObject[key] = json.Number(fmt.Sprintf("%d", intValue))
		}
	}
	cts.argumentObject[appdef.SystemField_ID] = id

	return cts
}

func (cts *CommandTestState) ArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	parts := strings.Split(path, "/")

	innerTree := cts.argumentObject
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}

		if i < len(parts)-1 {
			innerTree = putToArgumentObjectTree(innerTree, part)
			continue
		}

		innerTree = putToArgumentObjectTree(innerTree, part, keyValueList...)
		innerTree[appdef.SystemField_ID] = id
	}

	return cts
}

func (cts *CommandTestState) IntentSingletonInsert(fQName IFullQName, keyValueList ...any) ICommandRunner {
	cts.intentSingletonInsert(fQName, getSourceFileReference(2), keyValueList...)

	return cts
}

func (cts *CommandTestState) IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) ICommandRunner {
	cts.intentSingletonUpdate(fQName, getSourceFileReference(2), keyValueList...)

	return cts
}

func (cts *CommandTestState) IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.intentRecordInsert(fQName, id, getSourceFileReference(2), keyValueList...)

	return cts
}

func (cts *CommandTestState) IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.intentRecordUpdate(fQName, id, getSourceFileReference(2), keyValueList...)

	return cts
}

func (cts *CommandTestState) Run() {
	defer cts.recoverPanicInTestState()

	cts.putViewRecords()
	cts.putRecords()
	cts.putCudRows()
	cts.putArgument()

	cts.runExtensionFunc()

	cts.require()
}

// ProjectorTestState is a test state for projector testing
type ProjectorTestState struct {
	generalTestState

	rawEvent rawEvent
}

// NewProjectorTestState creates a new test state for projector testing
func NewProjectorTestState(t *testing.T, extensionFunc func()) *ProjectorTestState {
	pts := &ProjectorTestState{}
	pts.t = t
	pts.ctx = context.Background()
	pts.processorKind = ProcKind_Actualizer
	pts.secretReader = &secretReader{secrets: make(map[string][]byte)}
	pts.buildAppDef()
	// build state
	pts.IState = stateprovide.ProvideMockedActualizerStateFactory()(
		pts.ctx,
		IntentsLimit,
		func() istructs.IAppStructs { return pts.appStructs },
	)

	// initialize funcRunner and extensionFunc itself
	pts.funcRunner = &sync.Once{}
	pts.extensionFunc = extensionFunc
	pts.rawEvent = rawEvent{
		qName: appdef.NullQName,
		argumentObject: &coreutils.TestObject{
			Data: make(map[string]any),
		},
		unloggedArgumentObject: &coreutils.TestObject{
			Data: make(map[string]any),
		},
	}

	return pts
}

func (pts *ProjectorTestState) StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.stateRecord(fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateSingletonRecord(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.stateSingletonRecord(fQName, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventArgumentObject(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	setArgumentObject(pts.rawEvent.argumentObject, fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	setArgumentObjectRow(pts.rawEvent.argumentObject, path, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventUnloggedArgumentObject(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	setArgumentObject(pts.rawEvent.unloggedArgumentObject, fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventUnloggedArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	setArgumentObjectRow(pts.rawEvent.unloggedArgumentObject, path, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventCUD(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	if isODoc(fQName) {
		panic(fmt.Errorf("ODoc is not supported in the EventCUD method"))
	}

	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf(fmtMsgFailedToParseKeyValues, err))
	}

	pts.rawEvent.cuds = append(pts.rawEvent.cuds, &coreutils.TestObject{
		ID_:  id,
		Name: appdef.NewQName(getPackageLocalName(pts.appDef, fQName), fQName.Entity()),
		Data: keyValueMap,
	})

	return pts
}

func (pts *ProjectorTestState) IntentSingletonInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.intentSingletonInsert(fQName, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.intentSingletonUpdate(fQName, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.intentRecordInsert(fQName, id, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.intentRecordUpdate(fQName, id, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateCUDRow(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.cudRows = append(pts.cudRows, recordItem{
		entity:       fQName,
		id:           id,
		qName:        pts.getQNameFromFQName(fQName),
		keyValueList: keyValueList,
	})

	return pts
}

func (pts *ProjectorTestState) IntentViewInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.addRequiredItems(fQName, 0, false, true, true, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentViewUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.addRequiredItems(fQName, id, false, false, true, getSourceFileReference(2), keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateView(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	if !pts.isView(fQName) {
		panic("View method must be used for views only")
	}

	pts.viewRecords = append(pts.viewRecords, recordItem{
		entity:       fQName,
		qName:        pts.getQNameFromFQName(fQName),
		id:           id,
		isView:       true,
		keyValueList: keyValueList,
	})

	return pts
}

func (pts *ProjectorTestState) Run() {
	defer pts.recoverPanicInTestState()

	pts.putViewRecords()
	pts.putRecords()
	pts.putCudRows()
	pts.putEvent()

	pts.runExtensionFunc()

	pts.require()
}

//nolint:unused
func (pts *ProjectorTestState) putArgument() {
	if pts.rawEvent.argumentObject == nil {
		return
	}

	pts.putEvent()
}

func (pts *ProjectorTestState) putEvent() {
	mockedEventStorage, ok := pts.getMockedStorage(sys.Storage_Event)
	if !ok {
		panic("failed to get mocked event storage")
	}

	mockedEventObject := &coreutils.TestObject{
		Name:        pts.rawEvent.qName,
		Data:        pts.rawEvent.argumentObject.Data,
		Containers_: make(map[string][]*coreutils.TestObject),
	}
	mockedEventObject.Data["PLogOffset"] = pts.rawEvent.pLogOffset
	mockedEventObject.Data["HandlingPartition"] = pts.rawEvent.handlingPartition
	mockedEventObject.Data[sys.Storage_Event_Field_Workspace] = pts.rawEvent.Workspace()
	mockedEventObject.Data[sys.Storage_Event_Field_QName] = pts.rawEvent.qName
	mockedEventObject.Data[sys.Storage_CommandContext_Field_WLogOffset] = pts.rawEvent.wLogOffset

	for _, cud := range pts.rawEvent.cuds {
		cud.Data[appdef.SystemField_ID] = cud.ID_
		mockedEventObject.Containers_[sys.Storage_Event_Field_CUDs] = append(
			mockedEventObject.Containers_[sys.Storage_Event_Field_CUDs],
			&coreutils.TestObject{
				Name:        cud.Name,
				Data:        cud.Data,
				ID_:         cud.ID_,
				Parent_:     cud.Parent_,
				Containers_: cud.Containers_,
				IsNew_:      cud.IsNew_,
			},
		)
	}

	argQName := pts.rawEvent.argumentObject.Name
	pts.rawEvent.argumentObject.Data[appdef.SystemField_QName] = argQName
	// setting argument object
	mockedEventObject.Containers_[sys.Storage_Event_Field_ArgumentObject] = append(
		mockedEventObject.Containers_[sys.Storage_Event_Field_ArgumentObject],
		&coreutils.TestObject{
			Name:        argQName,
			Data:        pts.rawEvent.argumentObject.Data,
			Containers_: make(map[string][]*coreutils.TestObject),
		},
	)

	// writing an event object directly to the storage
	mockedEventStorage.PutRecord(0, mockedEventObject)
}

func (pts *ProjectorTestState) EventQName(fQName IFullQName) IProjectorRunner {
	pts.rawEvent.qName = appdef.NewQName(getPackageLocalName(pts.appDef, fQName), fQName.Entity())

	return pts
}

func (pts *ProjectorTestState) EventSynced(synced bool) IProjectorRunner {
	pts.rawEvent.synced = synced

	return pts
}

func (pts *ProjectorTestState) EventDeviceID(deviceID istructs.ConnectedDeviceID) IProjectorRunner {
	pts.rawEvent.deviceID = deviceID

	return pts
}

func (pts *ProjectorTestState) EventRegisteredAt(registeredAt time.Time) IProjectorRunner {
	pts.rawEvent.registeredAt = istructs.UnixMilli(registeredAt.UnixMilli())

	return pts
}

func (pts *ProjectorTestState) EventSyncedAt(syncedAt time.Time) IProjectorRunner {
	pts.rawEvent.syncedAt = istructs.UnixMilli(syncedAt.UnixMilli())

	return pts
}

func (pts *ProjectorTestState) EventWLogOffset(wLogOffset istructs.Offset) IProjectorRunner {
	pts.rawEvent.wLogOffset = wLogOffset

	return pts
}

func (pts *ProjectorTestState) EventPLogOffset(pLogOffset istructs.Offset) IProjectorRunner {
	pts.rawEvent.pLogOffset = pLogOffset

	return pts
}

func (pts *ProjectorTestState) EventWSID(wsid istructs.WSID) IProjectorRunner {
	pts.rawEvent.wsid = wsid

	return pts
}

func parseKeyValues(keyValues []any) (map[string]any, error) {
	if len(keyValues)%2 != 0 {
		return nil, errors.New("key-value list must be even")
	}

	result := make(map[string]any, len(keyValues)/2)
	for i := 0; i < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			return nil, errors.New("key must be a string")
		}

		value := keyValues[i+1]

		if intValue, ok := value.(int); ok {
			result[key] = json.Number(fmt.Sprintf("%d", intValue))
		} else {
			result[key] = value
		}
	}

	return result, nil
}

// putToArgumentObjectTree adds a value to the tree at the specified path part
// and returns the new tree or an error if the path part is not a valid key
func putToArgumentObjectTree(tree map[string]any, pathPart string, keyValueList ...any) map[string]any {
	if len(keyValueList) == 0 {
		newTree := make(map[string]any)
		tree[pathPart] = newTree

		return newTree
	}

	// check if path part is a valid key
	if _, ok := tree[pathPart]; !ok {
		tree[pathPart] = make([]any, 0)
	}

	newTree, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf(fmtMsgFailedToParseKeyValues, err))
	}

	// add key value map to the end of tree node
	tree[pathPart] = append(tree[pathPart].([]any), newTree)

	return newTree
}

// splitKeysFromValues splits map of key-value pairs into two maps:
// 1. map of keys
// 2. map of values
func splitKeysFromValues(entity IFullQName, m map[string]any) (mapOfKeys map[string]any, mapOfValues map[string]any) {
	mapOfKeys = make(map[string]any, len(m))
	mapOfValues = make(map[string]any, len(m))

	iView, ok := entity.(IView)
	if ok {
		keys := iView.Keys()
		for k, v := range m {
			if slices.Contains(keys, k) {
				mapOfKeys[k] = v
				continue
			}

			mapOfValues[k] = v
		}
	}

	return mapOfKeys, mapOfValues
}

func setArgumentObject(argumentObject *coreutils.TestObject, fQName IFullQName, id istructs.RecordID, keyValueList ...any) {
	argumentObject.ID_ = id
	argumentObject.Name = appdef.NewQName(fQName.PkgPath(), fQName.Entity())

	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf(fmtMsgFailedToParseKeyValues, err))
	}

	for key, value := range keyValueMap {
		v, valueExist := argumentObject.Data[key]
		if valueExist {
			panic(fmt.Errorf("key %s already exists in the argument object with value %v", key, v))
		}

		argumentObject.Data[key] = value
		if intValue, ok := value.(int); ok {
			argumentObject.Data[key] = json.Number(fmt.Sprintf("%d", intValue))
		}
	}

	argumentObject.Data[appdef.SystemField_ID] = id
}

func setArgumentObjectRow(argumentObject *coreutils.TestObject, path string, id istructs.RecordID, keyValueList ...any) {
	parts := strings.Split(path, "/")

	innerTree := argumentObject.Data
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}

		if i < len(parts)-1 {
			innerTree = putToArgumentObjectTree(innerTree, part)
			continue
		}

		innerTree = putToArgumentObjectTree(innerTree, part, keyValueList...)
		innerTree[appdef.SystemField_ID] = id
	}
}

// getSourceFileReference returns the file reference of the source file.
// The argument skip is the number of stack frames to ascend.
func getSourceFileReference(skip int) string {
	var fileRef string
	if _, file, line, ok := runtime.Caller(skip); ok {
		fileRef = fmt.Sprintf("%s:%d", file, line)
	}

	return fileRef
}

func getPackageLocalName(appDef appdef.IAppDef, fQName IFullQName) string {
	if fQName.PkgPath() == appdef.SysPackage {
		return fQName.PkgPath()
	}

	return appDef.PackageLocalName(fQName.PkgPath())
}

func isODoc(entity IFullQName) bool {
	_, ok := entity.(IODOc)
	return ok
}

type rawEvent struct {
	// abstract event fields
	qName                  appdef.QName
	cuds                   []*coreutils.TestObject
	argumentObject         *coreutils.TestObject
	unloggedArgumentObject *coreutils.TestObject
	registeredAt           istructs.UnixMilli
	deviceID               istructs.ConnectedDeviceID
	syncedAt               istructs.UnixMilli
	synced                 bool

	// raw event fields
	wLogOffset        istructs.Offset
	pLogOffset        istructs.Offset
	wsid              istructs.WSID
	handlingPartition istructs.PartitionID
}

func (re *rawEvent) QName() appdef.QName {
	return re.qName
}

func (re *rawEvent) ArgumentObject() istructs.IObject {
	return re.argumentObject
}

func (re *rawEvent) CUDs(cb func(istructs.ICUDRow) bool) {
	for _, cud := range re.cuds {
		if !cb(cud) {
			break
		}
	}
}

func (re *rawEvent) RegisteredAt() istructs.UnixMilli {
	return re.registeredAt
}

func (re *rawEvent) DeviceID() istructs.ConnectedDeviceID {
	return re.deviceID
}

func (re *rawEvent) Synced() bool {
	return re.synced
}

func (re *rawEvent) SyncedAt() istructs.UnixMilli {
	return re.syncedAt
}

func (re *rawEvent) ArgumentUnloggedObject() istructs.IObject {
	return re.unloggedArgumentObject
}

func (re *rawEvent) HandlingPartition() istructs.PartitionID {
	return re.handlingPartition
}

func (re *rawEvent) PLogOffset() istructs.Offset {
	return re.pLogOffset
}

func (re *rawEvent) WLogOffset() istructs.Offset {
	return re.wLogOffset
}

func (re *rawEvent) Workspace() istructs.WSID {
	return re.wsid
}
