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

// RecordIDs is global variable storing pointers to record ids created during test
var RecordIDs []*istructs.RecordID

// NewCommandTestState creates a new test state for command testing
func NewCommandTestState(t *testing.T, iCommand ICommand, extensionFunc func()) *TestState {
	const wsid = istructs.WSID(1)

	ts := &TestState{}
	ts.ctx = context.Background()
	ts.processorKind = ProcKind_CommandProcessor
	ts.commandWSID = wsid
	ts.secretReader = &secretReader{secrets: make(map[string][]byte)}

	// build appDef
	ts.buildAppDefNew(iCommand.PkgPath(), iCommand.WorkspaceDescriptor())
	ts.buildState(ProcKind_CommandProcessor)

	// initialize funcRunner and extensionFunc itself
	ts.funcRunner = &sync.Once{}
	ts.extensionFunc = extensionFunc
	// set test object
	ts.t = t
	// set cud builder function
	ts.setCudBuilder(iCommand, wsid)

	// set arguments for the command
	if len(iCommand.ArgumentEntity()) > 0 {
		ts.argumentType = appdef.NewFullQName(iCommand.ArgumentPkgPath(), iCommand.ArgumentEntity())
		ts.argumentObject = make(map[string]any)
	}

	RecordIDs = make([]*istructs.RecordID, 0)

	return ts
}

func (ts *TestState) setArgument() {
	if ts.argumentObject == nil {
		return
	}

	ts.PutEvent(ts.commandWSID, ts.argumentType, func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD) {
		argBuilder.FillFromJSON(ts.argumentObject)
	})
}

func (ts *TestState) setCudBuilder(wsItem IFullQName, wsid istructs.WSID) {
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

// buildAppDefNew alternative way of building IAppDef
func (ts *TestState) buildAppDefNew(wsPkgPath, wsDescriptorName string) {
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

func (ts *TestState) Record(fQName IFullQName, id int, keyValueList ...any) ICommandRunner {
	ts.recordItems = append(ts.recordItems, recordItem{
		entity:       fQName,
		id:           id,
		keyValueList: keyValueList,
	})

	recordID := istructs.RecordID(0)
	RecordIDs = append(RecordIDs, &recordID)

	return ts
}

func (ts *TestState) putRecords() {
	// put records into the state
	for _, item := range ts.recordItems {
		keyValueMap, err := parseKeyValues(item.keyValueList)
		require.NoError(ts.t, err)
		_, recordIDs := ts.PutRecords(ts.commandWSID, func(cud istructs.ICUD) {
			pkgAlias := ts.appDef.PackageLocalName(item.entity.PkgPath())

			fc := cud.Create(appdef.NewQName(pkgAlias, item.entity.Entity()))
			keyValueMap[appdef.SystemField_ID] = istructs.RecordID(item.id)
			fc.PutFromJSON(keyValueMap)
		})

		// add record ids to the state
		for i, recordID := range recordIDs {
			*RecordIDs[i] = recordID
		}
	}

	// clear record items after they are processed
	ts.recordItems = nil
}

func (ts *TestState) ArgumentObject(id int, keyValueList ...any) ICommandRunner {
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
func (ts *TestState) ArgumentObjectRow(path string, id int, keyValueList ...any) ICommandRunner {
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

func (ts *TestState) Run() ICommandRequire {
	// put records into the state
	ts.putRecords()
	// set argument
	ts.setArgument()
	// run extension function
	if ts.extensionFunc != nil {
		ts.funcRunner.Do(ts.extensionFunc)
	}
	return &intentAssertions{
		t:   ts.t,
		ctx: ts,
	}
}

// draft
func (ia *intentAssertions) SingletonInsert(fQName IFullQName, keyValueList ...any) {
	ia.singletonRequire(fQName, true, keyValueList...)
}

// draft
func (ia *intentAssertions) SingletonUpdate(fQName IFullQName, keyValueList ...any) {
	ia.singletonRequire(fQName, false, keyValueList...)
}

func (ia *intentAssertions) singletonRequire(fQName IFullQName, isInsertIntent bool, keyValueList ...any) {
	localPkgName := ia.ctx.appDef.PackageLocalName(fQName.PkgPath())
	localEntity := appdef.NewQName(localPkgName, fQName.Entity())
	kb, err := ia.ctx.IState.KeyBuilder(state.Record, localEntity)
	require.NoError(ia.t, err)

	keyValueMap, err := parseKeyValues(keyValueList)
	require.NoError(ia.t, err)

	state.PopulateKeys(kb, map[string]any{state.Field_IsSingleton: true})

	ia.vb = ia.ctx.IState.FindIntent(kb)
	require.NoError(ia.t, err)

	value := ia.vb.BuildValue()
	if value == nil {
		require.Fail(ia.t, "value builder does not support EqualValues operation")
		return
	}

	// check if we deal with Insert or Update intent
	if isInsertIntent {
		// sys.ID should be less than 65536 for Insert intents
		require.Less(ia.t, value.AsRecordID(appdef.SystemField_ID), istructs.FirstSingletonID)
	} else {
		// sys.ID should be greater or equal to 65536 for Update intents
		require.GreaterOrEqual(ia.t, value.AsRecordID(appdef.SystemField_ID), istructs.FirstSingletonID)
	}
	ia.EqualValues(keyValueMap)
}

// draft
func (ia *intentAssertions) RecordInsert(fQName IFullQName, id int, keyValueList ...any) {
	//TODO implement me

	// id < 65536
	panic("implement me")
}

// draft
func (ia *intentAssertions) RecordUpdate(fQName IFullQName, id int, keyValueList ...any) {
	// id > 65536

	//TODO implement me
	panic("implement me")
}

func (ia *intentAssertions) EqualValues(expectedValues map[string]any) {
	if ia.vb == nil {
		require.Fail(ia.t, "expected intent to exist")
		return
	}
	value := ia.vb.BuildValue()
	if value == nil {
		require.Fail(ia.t, "value builder does not support EqualValues operation")
		return
	}
	for expectedKey, expectedValue := range expectedValues {
		switch expectedValue.(type) {
		case int32:
			require.Equal(ia.t, expectedValue, value.AsInt32(expectedKey))
		case int64:
			require.Equal(ia.t, expectedValue, value.AsInt64(expectedKey))
		case float32:
			require.Equal(ia.t, expectedValue, value.AsFloat32(expectedKey))
		case float64:
			require.Equal(ia.t, expectedValue, value.AsFloat64(expectedKey))
		case []byte:
			require.Equal(ia.t, expectedValue, value.AsBytes(expectedKey))
		case string:
			require.Equal(ia.t, expectedValue, value.AsString(expectedKey))
		case bool:
			require.Equal(ia.t, expectedValue, value.AsBool(expectedKey))
		case appdef.QName:
			require.Equal(ia.t, expectedValue, value.AsQName(expectedKey))
		case istructs.IStateValue:
			require.Equal(ia.t, expectedValue, value.AsValue(expectedKey))
		default:
			require.Fail(ia.t, "unsupported value type")
		}
	}
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
