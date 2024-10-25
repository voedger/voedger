/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package actualizers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

type cud struct {
	isNew bool
	data  map[string]interface{}
}

func TestProjector_isAcceptable(t *testing.T) {
	require := require.New(t)
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
	qNameDoc1 := appdef.NewQName("my", "doc1")
	qNameDoc2 := appdef.NewQName("my", "doc2")

	adb := appdef.New()
	wsb := adb.AddWorkspace(appdef.NewQName(appdef.SysPackage, "workspace"))
	qNameODoc := appdef.NewQName(appdef.SysPackage, "oDoc")
	wsb.AddODoc(qNameODoc)
	appDef, err := adb.Build()
	require.NoError(err)

	tests := []struct {
		name             string
		triggeringQNames map[appdef.QName][]appdef.ProjectorEventKind
		cuds             map[appdef.QName]map[string]interface{}
		wantErrors       bool
		events           []istructs.IPLogEvent
		want             bool
	}{
		{
			name:       "Should accept any",
			wantErrors: false,
			events:     []istructs.IPLogEvent{newEvent(appdef.NullQName, appdef.NullQName, nil)},
			want:       true,
		},
		{
			name:       "Should accept error",
			wantErrors: true,
			events:     []istructs.IPLogEvent{newEvent(istructs.QNameForError, appdef.NullQName, nil)},
			want:       true,
		},
		{
			name:       "Should not accept error",
			wantErrors: false,
			events:     []istructs.IPLogEvent{newEvent(istructs.QNameForError, appdef.NullQName, nil)},
			want:       false,
		},
		{
			name:       "Should not accept sys.Corrupted",
			wantErrors: false,
			events:     []istructs.IPLogEvent{newEvent(istructs.QNameForCorruptedData, appdef.NullQName, nil)},
			want:       false,
		},
		{
			name:   "Should accept event",
			events: []istructs.IPLogEvent{newEvent(istructs.QNameCommand, appdef.NullQName, nil)},
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				istructs.QNameCommand: {appdef.ProjectorEventKind_Execute},
			},
			want: true,
		},
		{
			name:   "Should not accept event",
			events: []istructs.IPLogEvent{newEvent(istructs.QNameCommand, appdef.NullQName, nil)},
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				istructs.QNameQuery: {appdef.ProjectorEventKind_Execute},
			},
			want: false,
		},
		{
			name: "Should accept event args, doc of kind ODoc",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameODoc: {appdef.ProjectorEventKind_ExecuteWithParam},
			},
			events: []istructs.IPLogEvent{newEvent(appdef.NullQName, qNameODoc, nil)},
			want:   true,
		},
		{
			name: "Should accept event args, ODoc",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				istructs.QNameODoc: {appdef.ProjectorEventKind_ExecuteWithParam},
			},
			events: []istructs.IPLogEvent{newEvent(appdef.NullQName, qNameODoc, nil)},
			want:   true,
		},
		{
			name: "Should not accept event args",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				istructs.QNameQuery: {appdef.ProjectorEventKind_ExecuteWithParam},
			},
			events: []istructs.IPLogEvent{newEvent(appdef.NullQName, istructs.QNameCommand, nil)},
			want:   false,
		},
		{
			name: "Should accept AFTER INSERT",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Insert},
				qNameDoc2: {appdef.ProjectorEventKind_Insert},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc2: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc2: {isNew: true},
					qNameDoc2: {isNew: true},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER INSERT",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Insert},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc2: {isNew: true},
					qNameDoc2: {isNew: true},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: false},
					qNameDoc1: {isNew: false},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER UPDATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Update},
				qNameDoc2: {appdef.ProjectorEventKind_Update},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc2: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: false},
					qNameDoc2: {isNew: true},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER UPDATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Update},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc2: {isNew: false},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc1: {isNew: true},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER INSERT OR UPDATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Update, appdef.ProjectorEventKind_Insert},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: false},
					qNameDoc2: {isNew: true},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc2: {isNew: false},
				}),
			},
			want: true,
		},
		{
			name: "Should accept AFTER ACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Activate},
				qNameDoc2: {appdef.ProjectorEventKind_Activate},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					qNameDoc2: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER ACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Activate},
				qNameDoc2: {appdef.ProjectorEventKind_Activate},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
					qNameDoc2: {},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER DEACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Deactivate},
				qNameDoc2: {appdef.ProjectorEventKind_Deactivate},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
					qNameDoc2: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
			},
			want: true,
		},
		{
			name: "Should not accept AFTER DEACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Deactivate},
				qNameDoc2: {appdef.ProjectorEventKind_Deactivate},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc2: {},
				}),
			},
			want: false,
		},
		{
			name: "Should not accept AFTER ACTIVATE OR DEACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Deactivate, appdef.ProjectorEventKind_Activate},
				qNameDoc2: {appdef.ProjectorEventKind_Deactivate, appdef.ProjectorEventKind_Activate},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, nil),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
					qNameDoc2: {isNew: true},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {isNew: true},
					qNameDoc2: {},
				}),
			},
			want: false,
		},
		{
			name: "Should accept AFTER INSERT OR UPDATE OR ACTIVATE OR DEACTIVATE",
			triggeringQNames: map[appdef.QName][]appdef.ProjectorEventKind{
				qNameDoc1: {appdef.ProjectorEventKind_Deactivate, appdef.ProjectorEventKind_Activate, appdef.ProjectorEventKind_Insert, appdef.ProjectorEventKind_Update},
				qNameDoc2: {appdef.ProjectorEventKind_Deactivate, appdef.ProjectorEventKind_Activate, appdef.ProjectorEventKind_Insert, appdef.ProjectorEventKind_Update},
			},
			events: []istructs.IPLogEvent{
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {data: map[string]interface{}{appdef.SystemField_IsActive: true}},
					qNameDoc2: {data: map[string]interface{}{appdef.SystemField_IsActive: false}},
				}),
				newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					qNameDoc1: {},
					qNameDoc2: {isNew: true},
				}),
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, event := range test.events {
				require.Equal(test.want, isAcceptable(event, test.wantErrors, test.triggeringQNames, appDef, appdef.NewQName(appdef.SysPackage, "testProj")))
			}
		})
	}
}

func TestProjector_isAcceptableGlobalDocs(t *testing.T) {
	require := require.New(t)
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
	adb := appdef.New()
	wsb := adb.AddWorkspace(appdef.NewQName(appdef.SysPackage, "workspace"))
	qNameCDoc := appdef.NewQName(appdef.SysPackage, "cDoc")
	qNameWDoc := appdef.NewQName(appdef.SysPackage, "wDoc")
	qNameODoc := appdef.NewQName(appdef.SysPackage, "oDoc")
	qNameCRecord := appdef.NewQName(appdef.SysPackage, "cRecord")
	qNameWRecord := appdef.NewQName(appdef.SysPackage, "wRecord")
	qNameORecord := appdef.NewQName(appdef.SysPackage, "oRecord")
	wsb.AddCDoc(qNameCDoc)
	wsb.AddWDoc(qNameWDoc)
	wsb.AddODoc(qNameODoc)
	wsb.AddCRecord(qNameCRecord)
	wsb.AddWRecord(qNameWRecord)
	wsb.AddORecord(qNameORecord)
	appDef, err := adb.Build()
	require.NoError(err)

	// triggering global QName -> cud QNames from event -> should trigger or not
	tests := map[appdef.QName][]map[appdef.QName]bool{
		istructs.QNameCDoc: {
			{
				qNameCDoc:    true,
				qNameWDoc:    false,
				qNameODoc:    false,
				qNameCRecord: false,
				qNameWRecord: false,
				qNameORecord: false,
			},
		},
		istructs.QNameWDoc: {
			{
				qNameCDoc:    false,
				qNameWDoc:    true,
				qNameODoc:    false,
				qNameCRecord: false,
				qNameWRecord: false,
				qNameORecord: false,
			},
		},
		istructs.QNameODoc: {
			{
				qNameCDoc:    false,
				qNameWDoc:    false,
				qNameODoc:    true,
				qNameCRecord: false,
				qNameWRecord: false,
				qNameORecord: false,
			},
		},
		istructs.QNameCRecord: {
			{
				qNameCDoc:    true,
				qNameWDoc:    false,
				qNameODoc:    false,
				qNameCRecord: true,
				qNameWRecord: false,
				qNameORecord: false,
			},
		},
		istructs.QNameORecord: {
			{
				qNameCDoc:    false,
				qNameWDoc:    false,
				qNameODoc:    true,
				qNameCRecord: false,
				qNameWRecord: false,
				qNameORecord: true,
			},
		},
		istructs.QNameWRecord: {
			{
				qNameCDoc:    false,
				qNameWDoc:    true,
				qNameODoc:    false,
				qNameCRecord: false,
				qNameWRecord: true,
				qNameORecord: false,
			},
		},
	}

	for globalQName, eventCUDQNamesAndWant := range tests {
		for _, eventCUDQNameAndWant := range eventCUDQNamesAndWant {
			for eventCUDQName, want := range eventCUDQNameAndWant {
				event := newEvent(istructs.QNameCommandCUD, appdef.NullQName, map[appdef.QName]cud{
					eventCUDQName: {isNew: true},
				})
				require.Equal(want, isAcceptable(event, false, map[appdef.QName][]appdef.ProjectorEventKind{
					globalQName: {appdef.ProjectorEventKind_Insert},
				}, appDef, appdef.NewQName(appdef.SysPackage, "testProj")), fmt.Sprintf("global %s, cud %s", globalQName, eventCUDQName))
			}
		}
	}
}
