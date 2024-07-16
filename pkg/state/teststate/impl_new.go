/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package teststate

import (
	"context"
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
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/state"
	wsdescutil "github.com/voedger/voedger/pkg/utils/testwsdesc"
)

// CommandTestState is a test state for command testing
type CommandTestState struct {
	testState

	extensionFunc func()
	funcRunner    *sync.Once

	// recordItems is to store records
	recordItems []recordItem
	// requiredRecordItems is to store required items
	requiredRecordItems []recordItem
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
	// set cud builder function
	ts.setCudBuilder(iCommand, wsid)

	// set arguments for the command
	if len(iCommand.ArgumentEntity()) > 0 {
		ts.argumentType = appdef.NewFullQName(iCommand.ArgumentPkgPath(), iCommand.ArgumentEntity())
		ts.argumentObject = make(map[string]any)
	}

	return ts
}

func (ts *CommandTestState) setArgument() {
	if ts.argumentObject == nil {
		return
	}

	ts.testData[state.Field_ArgumentObject] = ts.argumentObject
}

func (ts *CommandTestState) setCudBuilder(wsItem IFullQName, wsid istructs.WSID) {
	localPkgName := ts.appDef.PackageLocalName(wsItem.PkgPath())
	if wsItem.PkgPath() == appdef.SysPackage {
		localPkgName = wsItem.PkgPath()
	}

	reb := ts.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: TestPartition,
			QName:             appdef.NewQName(localPkgName, wsItem.Entity()),
			WLogOffset:        0,
			PLogOffset:        ts.nextPLogOffs(),
		},
	})

	ts.cud = reb.CUDBuilder()
}

// buildAppDef alternative way of building IAppDef
func (ts *CommandTestState) buildAppDef(wsPkgPath, wsDescriptorName string) {
	compileResult, err := compile.Compile("..")
	if err != nil {
		panic(err)
	}

	ts.appDef = compileResult.AppDef

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, compileResult.AppDefBuilder)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	ts.appDef.Extensions(func(i appdef.IExtension) {
		if proj, ok := i.(appdef.IProjector); ok {
			if proj.Sync() {
				cfg.AddSyncProjectors(istructs.Projector{Name: i.QName()})
			} else {
				cfg.AddAsyncProjectors(istructs.Projector{Name: i.QName()})
			}
		} else if cmd, ok := i.(appdef.ICommand); ok {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmd.QName(), istructsmem.NullCommandExec))
		} else if q, ok := i.(appdef.IQuery); ok {
			cfg.Resources.Add(istructsmem.NewCommandFunction(q.QName(), istructsmem.NullCommandExec))
		}
	})

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

	ts.appStructs = structs
	ts.plogGen = istructsmem.NewIDGenerator()
	ts.wsOffsets = make(map[istructs.WSID]istructs.Offset)

	err = wsdescutil.CreateCDocWorkspaceDescriptorStub(ts.appStructs, TestPartition, ts.commandWSID, appdef.NewQName(filepath.Base(wsPkgPath), wsDescriptorName), ts.nextPLogOffs(), ts.nextWSOffs(ts.commandWSID))
	if err != nil {
		panic(err)
	}
}

func (ts *CommandTestState) record(fQName IFullQName, id int, isSingleton bool, keyValueList ...any) ICommandRunner {
	// TODO: error message must be understandable
	//Record(
	//	orm.Package_air.WSingleton_NextNumbers,
	//	65536,
	//	`NextPBillNumber`, nextNumber,
	//).
	qName := ts.getQNameFromFQName(fQName)

	// check if the record already exists
	slices.ContainsFunc(ts.recordItems, func(i recordItem) bool {
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

	ts.recordItems = append(ts.recordItems, recordItem{
		entity:       fQName,
		isSingleton:  isSingleton,
		id:           id,
		keyValueList: keyValueList,
	})

	return ts
}

func (ts *CommandTestState) Record(fQName IFullQName, id int, keyValueList ...any) ICommandRunner {
	isSingleton := ts.isSingletone(fQName)
	if isSingleton {
		panic("use SingletonRecord method for singletons")
	}

	return ts.record(fQName, id, isSingleton, keyValueList...)
}

// SingletonRecord adds a singleton record to the state
// Implemented in own method because of ID for singletons are generated under-the-hood and
// we can not insert singletons with our own IDs
func (ts *CommandTestState) SingletonRecord(fQName IFullQName, keyValueList ...any) ICommandRunner {
	isSingleton := ts.isSingletone(fQName)
	if !isSingleton {
		panic("use Record method for non-singleton entities")
	}
	qName := ts.getQNameFromFQName(fQName)

	// get real ID of the specific singleton
	id, err := ts.appStructs.Records().GetSingletonID(qName)
	if err != nil {
		panic(fmt.Errorf("failed to get singleton id: %w", err))
	}

	return ts.record(fQName, int(id), isSingleton, keyValueList...)
}

func (ts *CommandTestState) getQNameFromFQName(fQName IFullQName) appdef.QName {
	localPkgName := ts.appDef.PackageLocalName(fQName.PkgPath())
	return appdef.NewQName(localPkgName, fQName.Entity())
}

func (ts *CommandTestState) isSingletone(fQName IFullQName) bool {
	qName := ts.getQNameFromFQName(fQName)

	iSingleton := ts.appDef.Singleton(qName)
	if iSingleton != nil && iSingleton.Singleton() {
		return true
	}

	return false
}

func (ts *CommandTestState) putRecords() {
	// put records into the state
	for _, item := range ts.recordItems {
		pkgAlias := ts.appDef.PackageLocalName(item.entity.PkgPath())

		keyValueMap, err := parseKeyValues(item.keyValueList)
		require.NoError(ts.t, err, "failed to parse key values")

		keyValueMap[appdef.SystemField_QName] = appdef.NewQName(pkgAlias, item.entity.Entity()).String()
		keyValueMap[appdef.SystemField_ID] = istructs.RecordID(item.id)

		err = ts.appStructs.Records().PutJSON(ts.commandWSID, keyValueMap)
		require.NoError(ts.t, err)
	}

	// clear record items after they are processed
	ts.recordItems = nil
}

func (ts *CommandTestState) require() {
	// TODO: check requiring inexistent intents, error message must be understandable
	// check out required allIntents
	requiredKeys := make([]istructs.IStateKeyBuilder, 0, len(ts.requiredRecordItems))
	for _, item := range ts.requiredRecordItems {
		requiredKeys = append(requiredKeys, ts.keyBuilder(item))

		if item.isSingleton {
			ts.requireSingleton(item.entity, item.isNew, item.keyValueList...)
			continue
		}

		ts.requireRecord(item.entity, item.id, item.isNew, item.keyValueList...)
	}

	// gather all intents
	allIntents := make([]*intentItem, 0, ts.IState.IntentsCount())
	ts.IState.Intents(func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder, isNew bool) {
		allIntents = append(allIntents, &intentItem{
			key:   key,
			value: value,
			isNew: isNew,
		})
	})

	// workaround till the bug at the 308 line is fixed
	if len(allIntents) > len(requiredKeys) {
		require.Failf(ts.t, "the actual intent count exceeds expected one", "expected intent count: %d, actual count: %d", len(requiredKeys), len(allIntents))
	}

	//errList := make([]error, 0, len(allIntents))
	//// check out unexpected intents
	//for _, intent := range allIntents {
	//	found := false
	//	for _, requiredKey := range requiredKeys {
	//		if intent.key.Equals(requiredKey) {
	//			found = true
	//			continue
	//		}
	//	}
	//	if !found {
	//		// FIXME: runtime error: invalid memory address or nil pointer dereference in intent.String()
	//		errList = append(errList, fmt.Errorf("unexpected intent: %s", intent.String()))
	//	}
	//}
	//require.Emptyf(ts.t, errList, "unexpected intents: %w", errors.Join(errList...))

	// clear required record items after they are processed
	ts.requiredRecordItems = nil
}

func (ts *CommandTestState) ArgumentObject(id int, keyValueList ...any) ICommandRunner {
	keyValueMap, err := parseKeyValues(keyValueList)
	if err != nil {
		panic(fmt.Errorf("failed to parse key values: %w", err))
	}

	for key, value := range keyValueMap {
		v, valueExist := ts.argumentObject[key]
		if valueExist {
			panic(fmt.Errorf("key %s already exists in the argument object with value %v", key, v))
		}

		ts.argumentObject[key] = value
	}
	ts.argumentObject[appdef.SystemField_ID] = istructs.RecordID(id)

	return ts
}

// use id param as sys.ID
func (ts *CommandTestState) ArgumentObjectRow(path string, id int, keyValueList ...any) ICommandRunner {
	parts := strings.Split(path, "/")

	innerTree := ts.argumentObject
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}

		if i < len(parts)-1 {
			innerTree = putToArgumentObjectTree(innerTree, part)
			continue
		}

		innerTree = putToArgumentObjectTree(innerTree, part, keyValueList...)
		innerTree[appdef.SystemField_ID] = istructs.RecordID(id)
	}

	return ts
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

func (ts *CommandTestState) Run() {
	defer ts.require()

	ts.putRecords()
	ts.setArgument()

	// run extension function
	if ts.extensionFunc != nil {
		ts.funcRunner.Do(ts.extensionFunc)
	}
}

// draft
func (ts *CommandTestState) RequireSingletonInsert(fQName IFullQName, keyValueList ...any) ICommandRunner {
	return ts.addRequiredRecordItems(fQName, 0, true, true, keyValueList...)
}

// draft
func (ts *CommandTestState) RequireSingletonUpdate(fQName IFullQName, keyValueList ...any) ICommandRunner {
	return ts.addRequiredRecordItems(fQName, 0, true, false, keyValueList...)
}

func (ts *CommandTestState) addRequiredRecordItems(fQName IFullQName, id int, isSingleton, isNew bool, keyValueList ...any) ICommandRunner {
	ts.requiredRecordItems = append(ts.requiredRecordItems, recordItem{
		entity:       fQName,
		id:           id,
		isSingleton:  isSingleton,
		isNew:        isNew,
		keyValueList: keyValueList,
	})

	return ts
}

func (ts *CommandTestState) requireSingleton(fQName IFullQName, isInsertIntent bool, keyValueList ...any) {
	ts.requireIntent(fQName, 0, true, isInsertIntent, keyValueList...)
}

func (ts *CommandTestState) requireRecord(fQName IFullQName, id int, isInsertIntent bool, keyValueList ...any) {
	ts.requireIntent(fQName, id, false, isInsertIntent, keyValueList...)
}

// requireIntent checks if the intent exists in the state
// Parameters:
// fQName - full qname of the entity
// id - record id (unused for singletons)
// isSingletone - if the entity is a singleton
// isInsertIntent - if the intent is insert or update
// keyValueList - list of key-value pairs
func (ts *CommandTestState) requireIntent(
	fQName IFullQName,
	id int,
	isSingletone bool,
	isInsertIntent bool,
	keyValueList ...any,
) {
	localQName := ts.getQNameFromFQName(fQName)

	kb, err := ts.IState.KeyBuilder(state.Record, localQName)
	require.NoError(ts.t, err)

	if isSingletone {
		kb.PutBool(state.Field_IsSingleton, true)
	} else {
		kb.PutInt64(state.Field_ID, int64(id))
	}

	vb, isNew := ts.IState.FindIntentWithOpKind(kb)
	require.NoError(ts.t, err)

	value := vb.BuildValue()
	if value == nil {
		require.Fail(ts.t, "value builder does not support EqualValues operation")
		return
	}

	keyValueMap, err := parseKeyValues(keyValueList)
	require.NoError(ts.t, err)

	require.Equalf(ts.t, isInsertIntent, isNew, "%s: intent kind mismatch", localQName.String())
	ts.EqualValues(vb, keyValueMap)
}

// draft
func (ts *CommandTestState) RequireRecordInsert(fQName IFullQName, id int, keyValueList ...any) ICommandRunner {
	return ts.addRequiredRecordItems(fQName, id, false, true, keyValueList...)
}

// draft
func (ts *CommandTestState) RequireRecordUpdate(fQName IFullQName, id int, keyValueList ...any) ICommandRunner {
	return ts.addRequiredRecordItems(fQName, id, false, false, keyValueList...)
}

func (ts *CommandTestState) EqualValues(vb istructs.IStateValueBuilder, expectedValues map[string]any) {
	if vb == nil {
		require.Fail(ts.t, "expected value builder is nil")
		return
	}
	value := vb.BuildValue()
	if value == nil {
		require.Fail(ts.t, "value builder does not support EqualValues operation")
		return
	}
	for expectedKey, expectedValue := range expectedValues {
		switch t := expectedValue.(type) {
		case int:
			require.Equal(ts.t, int32(t), value.AsInt32(expectedKey))
		case int8:
			require.Equal(ts.t, int32(t), value.AsInt32(expectedKey))
		case int16:
			require.Equal(ts.t, int32(t), value.AsInt32(expectedKey))
		case int32:
			require.Equal(ts.t, t, value.AsInt32(expectedKey))
		case int64:
			require.Equal(ts.t, t, value.AsInt64(expectedKey))
		case float32:
			require.Equal(ts.t, t, value.AsFloat32(expectedKey))
		case float64:
			require.Equal(ts.t, t, value.AsFloat64(expectedKey))
		case []byte:
			require.Equal(ts.t, t, value.AsBytes(expectedKey))
		case string:
			require.Equal(ts.t, t, value.AsString(expectedKey))
		case bool:
			require.Equal(ts.t, t, value.AsBool(expectedKey))
		case appdef.QName:
			require.Equal(ts.t, t, value.AsQName(expectedKey))
		case istructs.IStateValue:
			require.Equal(ts.t, t, value.AsValue(expectedKey))
		default:
			require.Fail(ts.t, "unsupported value type")
		}
	}
}

func (ts *CommandTestState) keyBuilder(r recordItem) istructs.IStateKeyBuilder {
	localQName := ts.getQNameFromFQName(r.entity)

	kb, err := ts.IState.KeyBuilder(state.Record, localQName)
	require.NoError(ts.t, err, "IState.KeyBuilder: failed to create key builder")

	if r.isSingleton {
		kb.PutBool(state.Field_IsSingleton, true)
	} else {
		kb.PutInt64(state.Field_ID, int64(r.id))
	}

	return kb
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
		result[key] = value
	}

	return result, nil
}
