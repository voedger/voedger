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
)

func TestProjector_isAcceptable(t *testing.T) {
	newEvent := func(event, eventArgs appdef.QName) istructs.IPLogEvent {
		o := &mockObject{}
		o.On("QName").Return(eventArgs)
		e := &mockPLogEvent{}
		e.
			On("QName").Return(event).
			On("ArgumentObject").Return(o)
		return e
	}
	tests := []struct {
		name             string
		eventsFilter     []appdef.QName
		eventsArgsFilter []appdef.QName
		handleErrors     bool
		event            istructs.IPLogEvent
		want             bool
	}{
		{
			name:         "Should accept any",
			handleErrors: false,
			event:        newEvent(appdef.NullQName, appdef.NullQName),
			want:         true,
		},
		{
			name:         "Should accept error",
			handleErrors: true,
			event:        newEvent(istructs.QNameForError, appdef.NullQName),
			want:         true,
		},
		{
			name:         "Should not accept error",
			handleErrors: false,
			event:        newEvent(istructs.QNameForError, appdef.NullQName),
			want:         false,
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
				HandleErrors:     test.handleErrors,
			}

			require.Equal(t, test.want, isAcceptable(p, test.event))
		})
	}
}

type mockPLogEvent struct {
	istructs.IPLogEvent
	mock.Mock
}

func (e *mockPLogEvent) QName() appdef.QName { return e.Called().Get(0).(appdef.QName) }
func (e *mockPLogEvent) ArgumentObject() istructs.IObject {
	return e.Called().Get(0).(istructs.IObject)
}

type mockObject struct {
	istructs.IObject
	mock.Mock
}

func (o *mockObject) QName() appdef.QName { return o.Called().Get(0).(appdef.QName) }
