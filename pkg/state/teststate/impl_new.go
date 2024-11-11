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
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/sys"
)

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

func (gts *generalTestState) intentSingletonInsert(fQName IFullQName, keyValueList ...any) {
	gts.addRequiredRecordItems(fQName, 0, true, true, false, keyValueList...)
}

func (gts *generalTestState) intentSingletonUpdate(fQName IFullQName, keyValueList ...any) {
	gts.addRequiredRecordItems(fQName, 0, true, false, false, keyValueList...)
}

func (gts *generalTestState) intentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) {
	gts.addRequiredRecordItems(fQName, id, false, true, false, keyValueList...)
}

func (gts *generalTestState) intentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) {
	gts.addRequiredRecordItems(fQName, id, false, false, false, keyValueList...)
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
	qName := gts.getQNameFromFQName(fQName)

	// get real ID of the specific singleton
	id, err := gts.appStructs.Records().GetSingletonID(qName)
	if err != nil {
		panic(fmt.Errorf("failed to get singleton id: %w", err))
	}

	gts.record(fQName, id, isSingleton, keyValueList...)
}

func (gts *generalTestState) getQNameFromFQName(fQName IFullQName) appdef.QName {
	localPkgName := gts.appDef.PackageLocalName(fQName.PkgPath())
	return appdef.NewQName(localPkgName, fQName.Entity())
}

func (gts *generalTestState) isSingletone(fQName IFullQName) bool {
	qName := gts.getQNameFromFQName(fQName)

	iSingleton := gts.appDef.Singleton(qName)
	return iSingleton != nil && iSingleton.Singleton()
}

func (gts *generalTestState) isView(fQName IFullQName) bool {
	qName := gts.getQNameFromFQName(fQName)

	iView := gts.appDef.View(qName)
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

	for _, item := range gts.cudRows {
		kvMap, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, errMsgFailedToParseKeyValues)

		gts.PutEvent(
			gts.commandWSID,
			appdef.NewFullQName(item.entity.PkgPath(), item.entity.Entity()),
			func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD) {
				cud := cudBuilder.Create(gts.getQNameFromFQName(item.entity))
				cud.PutFromJSON(kvMap)
			},
		)
	}
}

func (gts *generalTestState) addRequiredRecordItems(
	fQName IFullQName,
	id istructs.RecordID,
	isSingleton, isNew, isView bool,
	keyValueList ...any,
) {
	gts.requiredRecordItems = append(gts.requiredRecordItems, recordItem{
		entity:       fQName,
		id:           id,
		isSingleton:  isSingleton,
		isNew:        isNew,
		isView:       isView,
		keyValueList: keyValueList,
	})
}

// recoverPanicInTestState must be called in defer to recover panic in the test state
func (gts *generalTestState) recoverPanicInTestState() {
	r := recover()
	if r != nil {
		require.Fail(gts.t, r.(error).Error())
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
		require.Fail(gts.t, "intent not found")
		return
	}

	m, err := parseKeyValues(r.keyValueList)
	require.NoError(gts.t, err, errMsgFailedToParseKeyValues)

	_, mapOfValues := splitKeysFromValues(r.entity, m)

	localQName := gts.getQNameFromFQName(r.entity)
	require.Equalf(gts.t, r.isNew, isNew, "%s: intent kind mismatch", localQName.String())
	gts.equalValues(vb, mapOfValues)
}

func (gts *generalTestState) equalValues(vb istructs.IStateValueBuilder, expectedValues map[string]any) {
	if vb == nil {
		require.Fail(gts.t, "expected value builder is nil")
		return
	}
	value := vb.BuildValue()
	if value == nil {
		require.Fail(gts.t, "value builder does not support equalValues operation")
		return
	}

	for expectedKey, expectedValue := range expectedValues {
		switch t := expectedValue.(type) {
		case int8:
			require.Equal(gts.t, int32(t), value.AsInt32(expectedKey))
		case int16:
			require.Equal(gts.t, int32(t), value.AsInt32(expectedKey))
		case int32:
			require.Equal(gts.t, t, value.AsInt32(expectedKey))
		case int64:
			require.Equal(gts.t, t, value.AsInt64(expectedKey))
		case int:
			require.Equal(gts.t, int64(t), value.AsInt64(expectedKey))
		case float32:
			require.Equal(gts.t, t, value.AsFloat32(expectedKey))
		case float64:
			require.Equal(gts.t, t, value.AsFloat64(expectedKey))
		case []byte:
			require.Equal(gts.t, t, value.AsBytes(expectedKey))
		case string:
			require.Equal(gts.t, t, value.AsString(expectedKey))
		case bool:
			require.Equal(gts.t, t, value.AsBool(expectedKey))
		case appdef.QName:
			require.Equal(gts.t, t, value.AsQName(expectedKey))
		case istructs.IStateValue:
			require.Equal(gts.t, t, value.AsValue(expectedKey))
		case json.Number:
			int64Value, err := t.Int64()
			require.NoError(gts.t, err)

			require.Equal(gts.t, int64Value, value.AsInt64(expectedKey))
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
		kb.PutInt64(sys.Storage_Record_Field_ID, int64(r.id)) // nolint G115
	case r.isView:
		m, err := parseKeyValues(r.keyValueList)
		require.NoError(gts.t, err, errMsgFailedToParseKeyValues)

		mapOfKeys, _ := splitKeysFromValues(r.entity, m)

		kb.PutFromJSON(mapOfKeys)
	}

	return kb
}

func (gts *generalTestState) record(fQName IFullQName, id istructs.RecordID, isSingleton bool, keyValueList ...any) {
	qName := gts.getQNameFromFQName(fQName)

	// check if the record already exists
	slices.ContainsFunc(gts.recordItems, func(i recordItem) bool {
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
		isSingleton:  isSingleton,
		id:           id,
		keyValueList: keyValueList,
	})
}

func (gts *generalTestState) putRecords() {
	// put records into the state
	for _, item := range gts.recordItems {
		pkgAlias := gts.appDef.PackageLocalName(item.entity.PkgPath())

		keyValueMap, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, errMsgFailedToParseKeyValues)

		keyValueMap[appdef.SystemField_QName] = appdef.NewQName(pkgAlias, item.entity.Entity()).String()
		keyValueMap[appdef.SystemField_ID] = item.id

		err = gts.appStructs.Records().PutJSON(gts.commandWSID, keyValueMap)
		require.NoError(gts.t, err)
	}

	// clear record items after they are processed
	gts.recordItems = nil
}

func (gts *generalTestState) putViewRecords() {
	for _, item := range gts.viewRecords {
		m, err := parseKeyValues(item.keyValueList)
		require.NoError(gts.t, err, errMsgFailedToParseKeyValues)

		mapOfKeys, mapOfValues := splitKeysFromValues(item.entity, m)

		gts.PutView(
			gts.commandWSID,
			appdef.NewFullQName(
				item.entity.PkgPath(),
				item.entity.Entity(),
			),
			func(key istructs.IKeyBuilder, value istructs.IValueBuilder) {
				key.PutFromJSON(mapOfKeys)
				value.PutFromJSON(mapOfValues)
			},
		)
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

	ts := &CommandTestState{}

	ts.testData = make(map[string]any)
	// set test object
	ts.t = t

	ts.ctx = context.Background()
	ts.processorKind = ProcKind_CommandProcessor
	ts.commandWSID = wsid
	ts.secretReader = &secretReader{secrets: make(map[string][]byte)}

	// build appDef
	ts.buildAppDef(iCommand.PkgPath(), iCommand.WorkspaceDescriptor())
	ts.buildState(ProcKind_CommandProcessor)

	// initialize funcRunner and extensionFunc itself
	ts.funcRunner = &sync.Once{}
	ts.extensionFunc = extensionFunc

	ts.argumentObject = make(map[string]any)
	if iCommand != nil {
		// set cud builder function
		ts.setCudBuilder(iCommand, wsid)

		// set arguments for the command
		if len(iCommand.ArgumentEntity()) > 0 {
			ts.argumentType = appdef.NewFullQName(iCommand.ArgumentPkgPath(), iCommand.ArgumentEntity())
		}
	}

	return ts
}

func (cts *CommandTestState) StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.generalTestState.stateRecord(fQName, id, keyValueList...)

	return cts
}

func (cts *CommandTestState) StateSingletonRecord(fQName IFullQName, keyValueList ...any) ICommandRunner {
	cts.generalTestState.stateSingletonRecord(fQName, keyValueList...)

	return cts
}

func (cts *CommandTestState) putArgument() {
	if cts.argumentObject == nil {
		return
	}

	cts.testData[sys.Storage_CommandContext_Field_ArgumentObject] = cts.argumentObject
}

func (cts *CommandTestState) setCudBuilder(wsItem IFullQName, wsid istructs.WSID) {
	localPkgName := cts.appDef.PackageLocalName(wsItem.PkgPath())
	if wsItem.PkgPath() == appdef.SysPackage {
		localPkgName = wsItem.PkgPath()
	}

	reb := cts.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: TestPartition,
			QName:             appdef.NewQName(localPkgName, wsItem.Entity()),
			WLogOffset:        0,
			PLogOffset:        cts.nextPLogOffs(),
		},
	})

	cts.cud = reb.CUDBuilder()
}

// buildAppDef alternative way of building IAppDef
func (cts *CommandTestState) buildAppDef(wsPkgPath, wsDescriptorName string) {
	compileResult, err := compile.Compile("..")
	if err != nil {
		panic(err)
	}

	cts.appDef = compileResult.AppDef

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, compileResult.AppDefBuilder)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	for ext := range cts.appDef.Extensions {
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

	asf := mem.Provide()
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider,
	)

	structs, err := prov.BuiltIn(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}

	cts.appStructs = structs
	cts.plogGen = istructsmem.NewIDGenerator()
	cts.wsOffsets = make(map[istructs.WSID]istructs.Offset)

	err = wsdescutil.CreateCDocWorkspaceDescriptorStub(
		cts.appStructs,
		TestPartition,
		cts.commandWSID,
		appdef.NewQName(filepath.Base(wsPkgPath), wsDescriptorName),
		cts.nextPLogOffs(),
		cts.nextWSOffs(cts.commandWSID),
	)
	if err != nil {
		panic(err)
	}
}

func (cts *CommandTestState) ArgumentObject(id istructs.RecordID, keyValueList ...any) ICommandRunner {
	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf("failed to parse key values: %w", err))
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
	cts.intentSingletonInsert(fQName, 0, true, true, false, keyValueList)

	return cts
}

func (cts *CommandTestState) IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) ICommandRunner {
	cts.intentSingletonUpdate(fQName, 0, true, false, false, keyValueList)

	return cts
}

func (cts *CommandTestState) IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.intentRecordInsert(fQName, id, false, true, false, keyValueList)

	return cts
}

func (cts *CommandTestState) IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner {
	cts.intentRecordUpdate(fQName, id, false, false, false, keyValueList)

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
}

// NewProjectorTestState creates a new test state for projector testing
func NewProjectorTestState(t *testing.T, iProjector IProjector, extensionFunc func()) *ProjectorTestState {
	ts := &ProjectorTestState{}
	ts.ctx = context.Background()
	ts.processorKind = ProcKind_Actualizer
	ts.secretReader = &secretReader{secrets: make(map[string][]byte)}
	ts.buildAppDef(iProjector.PkgPath(), iProjector.WorkspaceDescriptor())
	ts.buildState(ProcKind_Actualizer)
	// initialize funcRunner and extensionFunc itself
	ts.funcRunner = &sync.Once{}
	ts.extensionFunc = extensionFunc

	return ts
}

func (pts *ProjectorTestState) StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.stateRecord(fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateSingletonRecord(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.stateSingletonRecord(fQName, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) EventArgumentObject(id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf("failed to parse key values: %w", err))
	}

	for key, value := range keyValueMap {
		v, valueExist := pts.argumentObject[key]
		if valueExist {
			panic(fmt.Errorf("key %s already exists in the argument object with value %v", key, v))
		}

		pts.argumentObject[key] = value
		if intValue, ok := value.(int); ok {
			pts.argumentObject[key] = json.Number(fmt.Sprintf("%d", intValue))
		}
	}

	pts.argumentObject[appdef.SystemField_ID] = id

	return pts
}

func (pts *ProjectorTestState) EventArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	parts := strings.Split(path, "/")

	innerTree := pts.argumentObject
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

	return pts
}

func (pts *ProjectorTestState) IntentSingletonInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.intentSingletonInsert(fQName, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.intentSingletonUpdate(fQName, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.intentRecordInsert(fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.intentRecordUpdate(fQName, id, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateCUDRow(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.cudRows = append(pts.cudRows, recordItem{
		entity:       fQName,
		id:           id,
		keyValueList: keyValueList,
	})

	return pts
}

func (pts *ProjectorTestState) IntentViewInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner {
	pts.addRequiredRecordItems(fQName, 0, false, true, true, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) IntentViewUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	pts.addRequiredRecordItems(fQName, id, false, false, true, keyValueList...)

	return pts
}

func (pts *ProjectorTestState) StateView(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner {
	if !pts.isView(fQName) {
		panic("View method must be used for views only")
	}

	pts.viewRecords = append(pts.viewRecords, recordItem{
		entity:       fQName,
		id:           id,
		isView:       true,
		keyValueList: keyValueList,
	})

	return pts
}

func (pts *ProjectorTestState) EventOffset(offset istructs.Offset) IProjectorRunner {
	pts.wsOffsets[pts.commandWSID] = offset

	return pts
}

func (pts *ProjectorTestState) Run() {
	defer pts.recoverPanicInTestState()

	pts.putViewRecords()
	pts.putRecords()
	pts.putCudRows()
	pts.putArgument()

	pts.runExtensionFunc()

	pts.require()
}

func (pts *ProjectorTestState) putArgument() {
	if pts.argumentObject == nil {
		return
	}

	pts.PutEvent(
		pts.commandWSID,
		pts.argumentType,
		func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD) {
			argBuilder.FillFromJSON(pts.argumentObject)
		},
	)
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
	_, ok := tree[pathPart]
	if !ok {
		tree[pathPart] = make([]any, 0)
	}

	newTree, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf("failed to parse key values: %w", err))
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
