/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package actualizers_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/set"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors/actualizers"
)

type cud struct {
	isNew bool
	data  map[string]interface{}
}

func Test_ProjectorEvent(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("my", "workspace")
	prjName := appdef.NewQName("my", "projector")
	cmdName := appdef.NewQName("my", "command")
	oDocName := appdef.NewQName("my", "ODoc")
	objName := appdef.NewQName("my", "object")
	cDocName := appdef.NewQName("my", "CDoc")
	wDocName := appdef.NewQName("my", "WDoc")

	newProjector := func(ops appdef.OperationsSet, flt appdef.IFilter, wantErrors bool) appdef.IProjector {
		adb := appdef.New()
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddODoc(oDocName)
		_ = wsb.AddObject(objName)
		_ = wsb.AddCDoc(cDocName)
		_ = wsb.AddWDoc(wDocName)
		prj := wsb.AddProjector(prjName)
		prj.Events().Add(ops.AsArray(), flt)
		if wantErrors {
			prj.SetWantErrors()
		}
		return appdef.Projector(adb.MustBuild().Type, prjName)
	}
	newEvent := func(eventQName, eventArgsQName appdef.QName, cuds map[appdef.QName]cud) istructs.IPLogEvent {
		event := &coreutils.MockPLogEvent{}
		event.On("QName").Return(eventQName)
		event.On("ArgumentObject").Return(&coreutils.TestObject{Name: eventArgsQName})
		event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(cb istructs.ICUDRow) bool)
			for cudQName, cud := range cuds {
				cudRow := &coreutils.TestObject{
					Name:   cudQName,
					Data:   cud.data,
					IsNew_: cud.isNew,
				}
				if !cb(cudRow) {
					break
				}
			}
		})
		return event
	}

	type testEvent struct {
		event istructs.IPLogEvent
		want  bool
	}
	tests := []struct {
		name   string
		prj    appdef.IProjector
		events []testEvent
	}{
		{
			name: "Should accept any",
			prj:  newProjector(appdef.ProjectorOperations, filter.True(), false),
			events: []testEvent{
				{newEvent(appdef.NullQName, appdef.NullQName, nil), true},
			},
		},
		{
			name: "Should accept error",
			prj:  newProjector(appdef.ProjectorOperations, filter.True(), true),
			events: []testEvent{
				{newEvent(istructs.QNameForError, appdef.NullQName, nil), true},
			},
		},
		{
			name: "Should not accept error",
			prj:  newProjector(appdef.ProjectorOperations, filter.True(), false),
			events: []testEvent{
				{newEvent(istructs.QNameForError, appdef.NullQName, nil), false},
			},
		},
		{
			name: "Should not accept sys.Corrupted",
			prj:  newProjector(appdef.ProjectorOperations, filter.True(), false),
			events: []testEvent{
				{newEvent(istructs.QNameForCorruptedData, appdef.NullQName, nil), false},
			},
		},
		{
			name: "Should accept my.Command",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.Types(wsName, appdef.TypeKind_Command), false),
			events: []testEvent{
				{newEvent(cmdName, appdef.NullQName, nil), true},
				{newEvent(oDocName, appdef.NullQName, nil), false}, // not a command
			},
		},
		{
			name: "Should not accept my.Command",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.Types(wsName, appdef.TypeKind_Query), false),
			events: []testEvent{
				{newEvent(cmdName, appdef.NullQName, nil), false}, // not a query
			},
		},
		{
			name: "Should accept event my.ODoc",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.QNames(oDocName), false),
			events: []testEvent{
				{newEvent(oDocName, appdef.NullQName, nil), true},
				{newEvent(cmdName, appdef.NullQName, nil), false}, // not my.ODoc
			},
		},
		{
			name: "Should accept my.Command with argument",
			prj:  newProjector(set.From(appdef.OperationKind_ExecuteWithParam), filter.QNames(oDocName, objName), false),
			events: []testEvent{
				{newEvent(cmdName, oDocName, nil), true},
				{newEvent(cmdName, objName, nil), true},
				{newEvent(cmdName, wDocName, nil), false}, // not my.ODoc or my.object
			},
		},
		{
			name: "Should accept AFTER INSERT",
			prj:  newProjector(set.From(appdef.OperationKind_Insert), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: true},
					wDocName: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					wDocName: {isNew: true},
					wDocName: {isNew: true},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER INSERT",
			prj:  newProjector(set.From(appdef.OperationKind_Insert), filter.QNames(cDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					wDocName: {isNew: true},
					cDocName: {isNew: false},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER UPDATE",
			prj:  newProjector(set.From(appdef.OperationKind_Update), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: true},
					wDocName: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: false},
					wDocName: {isNew: true},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER UPDATE",
			prj:  newProjector(set.From(appdef.OperationKind_Update), filter.QNames(cDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: true},
					wDocName: {isNew: false},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER INSERT OR UPDATE",
			prj:  newProjector(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update), filter.QNames(cDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: true},
				}),
			},
			want: true,
		},
		{
			name: "Should accept AFTER ACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Activate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					wDocName: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER ACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Activate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
					wDocName: {},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER DEACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Deactivate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					wDocName: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER DEACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Deactivate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					wDocName: {},
				}),
			},
			want: false,
		},
		{
			name: "Should not accept AFTER ACTIVATE OR DEACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Activate, appdef.OperationKind_Deactivate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
					wDocName: {isNew: true},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {isNew: true},
					wDocName: {},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER INSERT OR UPDATE OR ACTIVATE OR DEACTIVATE",
			prj:  newProjector(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate), filter.QNames(cDocName, wDocName), false),
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					cDocName: {},
					wDocName: {isNew: true},
				}),
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, event := range test.events {
				require.Equal(event.want, actualizers.ProjectorEvent(test.prj, event.IPLogEvent))
			}
		})
	}
}
