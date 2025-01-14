/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package actualizers_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
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
	queryName := appdef.NewQName("my", "query")
	oDocName := appdef.NewQName("my", "ODoc")
	objName := appdef.NewQName("my", "object")
	cDocName := appdef.NewQName("my", "CDoc")
	wDocName := appdef.NewQName("my", "WDoc")
	tagName := appdef.NewQName("my", "CDocTag")

	newProjector := func(ops appdef.OperationsSet, flt appdef.IFilter, wantErrors bool) appdef.IProjector {
		adb := builder.New()
		sysWsb := adb.AlterWorkspace(appdef.SysWorkspaceQName)
		_ = sysWsb.AddCommand(istructs.QNameCommandCUD) // should be in ancestor

		wsb := adb.AddWorkspace(wsName)
		wsb.AddTag(tagName)
		//_ = wsb.AddCommand(istructs.QNameCommandCUD) // should be in ancestor
		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)
		_ = wsb.AddODoc(oDocName)
		_ = wsb.AddObject(objName)
		wsb.AddCDoc(cDocName).SetTag(tagName)
		_ = wsb.AddWDoc(wDocName)
		prj := wsb.AddProjector(prjName)
		prj.Events().Add(ops.AsArray(), flt)
		if wantErrors {
			prj.SetWantErrors()
		}
		return appdef.Projector(adb.MustBuild().Type, prjName)
	}

	newProjectorOnAll := func(wantErrors bool) appdef.IProjector {
		return newProjector(appdef.ProjectorOperations, filter.True(), wantErrors)
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
		name string
		plog istructs.IPLogEvent
		want bool
	}
	tests := []struct {
		name   string
		prj    appdef.IProjector
		events []testEvent
	}{
		{
			name: "projector ON ALL",
			prj:  newProjectorOnAll(true),
			events: []testEvent{
				{"accept sys.error", newEvent(istructs.QNameForError, appdef.NullQName, nil), true},
				{"reject sys.corrupted", newEvent(istructs.QNameForCorruptedData, appdef.NullQName, nil), false},
			},
		},
		{
			name: "projector ON ALL except sys.error",
			prj:  newProjectorOnAll(false),
			events: []testEvent{
				{"accept my.cmd", newEvent(cmdName, appdef.NullQName, nil), true},
				{"accept sys.CUD", newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil), true},
				{"reject sys.error", newEvent(istructs.QNameForError, appdef.NullQName, nil), false},
				{"reject sys.corrupted", newEvent(istructs.QNameForCorruptedData, appdef.NullQName, nil), false},
			},
		},
		{
			name: "projector ON EXECUTE ALL COMMANDS",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.Types(appdef.TypeKind_Command), false),
			events: []testEvent{
				{"accept my.command", newEvent(cmdName, appdef.NullQName, nil), true},
				{"accept sys.CUD", newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil), true},
				{"reject my.ODoc", newEvent(oDocName, appdef.NullQName, nil), false}, // not a command
			},
		},
		{
			name: "projector ON EXECUTE ALL COMMANDS FROM my.workspace",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.WSTypes(wsName, appdef.TypeKind_Command), false),
			events: []testEvent{
				{"accept my.command", newEvent(cmdName, appdef.NullQName, nil), true},
				{"accept sys.CUD", newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil), false}, // not from my.workspace
				{"reject my.ODoc", newEvent(oDocName, appdef.NullQName, nil), false},                 // not a command
			},
		},
		{
			name: "projector ON EXECUTE ALL QUERIES",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.Types(appdef.TypeKind_Query), false),
			events: []testEvent{
				{"reject my.command", newEvent(cmdName, appdef.NullQName, nil), false}, // not a query
			},
		},
		{
			name: "projector ON EXECUTE my.ODoc",
			prj:  newProjector(set.From(appdef.OperationKind_Execute), filter.QNames(oDocName), false),
			events: []testEvent{
				{"accept my.ODoc", newEvent(oDocName, appdef.NullQName, nil), true},
				{"reject my.command", newEvent(cmdName, appdef.NullQName, nil), false}, // not my.ODoc
			},
		},
		{
			name: "projector ON EXECUTE WITH PARAM my.ODoc OR my.object",
			prj:  newProjector(set.From(appdef.OperationKind_ExecuteWithParam), filter.QNames(oDocName, objName), false),
			events: []testEvent{
				{"accept my.ODoc", newEvent(cmdName, oDocName, nil), true},
				{"accept my.object", newEvent(cmdName, objName, nil), true},
				{"reject my.WDoc", newEvent(cmdName, wDocName, nil), false}, // not my.ODoc or my.object
			},
		},
		{
			name: "projector AFTER INSERT my.CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Insert), filter.QNames(cDocName), false),
			events: []testEvent{
				{"accept insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					true},
				{"reject update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					false}, // not INSERT
				{"reject insert my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: true},
					}),
					false}, // not my.CDoc
			},
		},
		{
			name: "projector AFTER INSERT ALL CDocs",
			prj:  newProjector(set.From(appdef.OperationKind_Insert), filter.Types(appdef.TypeKind_CDoc), false),
			events: []testEvent{
				{"accept insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					true},
				{"reject insert my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: true},
					}),
					false}, // not CDoc
			},
		},
		{
			name: "projector AFTER UPDATE my.CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Update), filter.QNames(cDocName), false),
			events: []testEvent{
				{"accept update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					true},
				{"reject insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					false}, // not UPDATE
				{"reject update my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: false},
					}),
					false}, // not my.CDoc
			},
		},
		{
			name: "projector AFTER UPDATE ALL CDocs",
			prj:  newProjector(set.From(appdef.OperationKind_Update), filter.Types(appdef.TypeKind_CDoc), false),
			events: []testEvent{
				{"accept update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					true},
				{"reject update my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: false},
					}),
					false}, // not CDoc
			},
		},
		{
			name: "projector AFTER INSERT OR UPDATE my.CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update), filter.QNames(cDocName), false),
			events: []testEvent{
				{"accept insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					true},
				{"accept update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					true},
				{"reject insert or update my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: true},
						wDocName: {isNew: false},
					}),
					false}, // not my.CDoc
			},
		},
		{
			name: "projector AFTER INSERT OR UPDATE ALL CDocs",
			prj:  newProjector(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update), filter.Types(appdef.TypeKind_CDoc), false),
			events: []testEvent{
				{"accept insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					true},
				{"accept update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					true},
				{"reject insert or update my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: true},
						wDocName: {isNew: false},
					}),
					false}, // not CDoc
			},
		},
		{
			name: "projector AFTER ACTIVATE my.CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Activate), filter.QNames(cDocName), false),
			events: []testEvent{
				{"accept activate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					true},
				{"reject insert or update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
						cDocName: {data: map[string]interface{}{"field1": 0}},
					}),
					false}, // INSERT and UPDATE, but not ACTIVATE
				{"reject deactivate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					false}, // DEACTIVATE
				{"reject activate my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					false}, // not my.CDoc
			},
		},
		{
			name: "projector AFTER DEACTIVATE my.CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Deactivate), filter.QNames(cDocName), false),
			events: []testEvent{
				{"accept deactivate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					true},
				{"reject insert or update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
						cDocName: {data: map[string]interface{}{"field1": 0}},
					}),
					false}, // INSERT and UPDATE, but not DEACTIVATE
				{"reject activate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					false}, // ACTIVATE
				{"reject deactivate my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					false}, // not my.CDoc
			},
		},
		{
			name: "projector AFTER ACTIVATE OR DEACTIVATE ALL CDoc",
			prj:  newProjector(set.From(appdef.OperationKind_Activate, appdef.OperationKind_Deactivate), filter.Types(appdef.TypeKind_CDoc), false),
			events: []testEvent{
				{"accept activate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					true},
				{"accept deactivate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					true},
				{"reject insert or update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
						cDocName: {data: map[string]interface{}{"field1": 0}},
					}),
					false}, // INSERT and UPDATE, but not (ACTIVATE or DEACTIVATE)
				{"reject activate or deactivate my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					false}, // not CDoc
			},
		},
		{
			name: "projector AFTER INSERT OR UPDATE OR ACTIVATE OR DEACTIVATE ALL WITH TAG my.CDocTag",
			prj:  newProjector(set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate), filter.Tags(tagName), false),
			events: []testEvent{
				{"accept insert my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: true},
					}),
					true},
				{"accept update my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {isNew: false},
					}),
					true},
				{"accept activate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					}),
					true},
				{"accept deactivate my.CDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						cDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					true},
				{"reject insert, update, activate or deactivate my.WDoc",
					newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
						wDocName: {isNew: true},
						wDocName: {isNew: false},
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
						wDocName: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					}),
					false}, // not with tag my.CDocTag
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, event := range test.events {
				t.Run(event.name, func(t *testing.T) {
					require.Equal(event.want, actualizers.ProjectorEvent(test.prj, event.plog))
				})
			}
		})
	}
}
