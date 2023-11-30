/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package projectors

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestProjector_isAcceptable(t *testing.T) {
	newEvent := func(eventQName, eventArgsQName appdef.QName, cuds map[appdef.QName]map[string]interface{}) istructs.IPLogEvent {
		event := &coreutils.MockPLogEvent{}
		event.On("QName").Return(eventQName)
		event.On("ArgumentObject").Return(eventArgsQName)
		event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(cb istructs.ICUDRow))
			for cudQName, cudData := range cuds {
				cudRow := &coreutils.TestObject{
					Name: cudQName,
					Data: cudData,
				}
				cb(cudRow)

			}
		})

		// o := &coreutils.MockPLogEvent{}
		// o.On("QName").Return(eventArgs)
		// e := &mockPLogEvent{}
		// e.
		// 	On("QName").Return(event).
		// 	On("ArgumentObject").Return(o)
		// return e
	}
	tests := []struct {
		name             string
		triggeringQNames map[appdef.QName][]appdef.ProjectorEventKind
		wantErrors       bool
		event            istructs.IPLogEvent
		want             bool
	}{
		{
			name:       "Should accept any",
			wantErrors: false,
			event:      newEvent(appdef.NullQName, appdef.NullQName),
			want:       true,
		},
		{
			name:       "Should accept error",
			wantErrors: true,
			event:      newEvent(istructs.QNameForError, appdef.NullQName),
			want:       true,
		},
		{
			name:       "Should not accept error",
			wantErrors: false,
			event:      newEvent(istructs.QNameForError, appdef.NullQName),
			want:       false,
		},
		{
			name:         "Should accept event",
			eventsFilter: []appdef.QName{istructs.QNameCommand},
			event:        newEvent(istructs.QNameCommand, appdef.NullQName),
			want:         true,
		},
		{
			name:         "Should not accept event",
			eventsFilter: []appdef.QName{istructs.QNameQuery},
			event:        newEvent(istructs.QNameCommand, appdef.NullQName),
			want:         false,
		},
		{
			name:             "Should accept event args",
			eventsArgsFilter: []appdef.QName{istructs.QNameCommand},
			event:            newEvent(appdef.NullQName, istructs.QNameCommand),
			want:             true,
		},
		{
			name:             "Should not accept event args",
			eventsArgsFilter: []appdef.QName{istructs.QNameQuery},
			event:            newEvent(appdef.NullQName, istructs.QNameCommand),
			want:             false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := istructs.Projector{
				EventsFilter:     test.eventsFilter,
				EventsArgsFilter: test.eventsArgsFilter,
				HandleErrors:     test.wantErrors,
			}

			require.Equal(t, test.want, isAcceptable(p, test.event))
		})
	}
}
