/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package projectors

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestProjector_isAcceptable(t *testing.T) {
	newEvent := func(event, eventArgs schemas.QName) istructs.IPLogEvent {
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
		eventsFilter     []schemas.QName
		eventsArgsFilter []schemas.QName
		handleErrors     bool
		event            istructs.IPLogEvent
		want             bool
	}{
		{
			name:         "Should accept any",
			handleErrors: false,
			event:        newEvent(schemas.NullQName, schemas.NullQName),
			want:         true,
		},
		{
			name:         "Should accept error",
			handleErrors: true,
			event:        newEvent(istructs.QNameForError, schemas.NullQName),
			want:         true,
		},
		{
			name:         "Should not accept error",
			handleErrors: false,
			event:        newEvent(istructs.QNameForError, schemas.NullQName),
			want:         false,
		},
		{
			name:         "Should accept event",
			eventsFilter: []schemas.QName{istructs.QNameCommand},
			event:        newEvent(istructs.QNameCommand, schemas.NullQName),
			want:         true,
		},
		{
			name:         "Should not accept event",
			eventsFilter: []schemas.QName{istructs.QNameQuery},
			event:        newEvent(istructs.QNameCommand, schemas.NullQName),
			want:         false,
		},
		{
			name:             "Should accept event args",
			eventsArgsFilter: []schemas.QName{istructs.QNameCommand},
			event:            newEvent(schemas.NullQName, istructs.QNameCommand),
			want:             true,
		},
		{
			name:             "Should not accept event args",
			eventsArgsFilter: []schemas.QName{istructs.QNameQuery},
			event:            newEvent(schemas.NullQName, istructs.QNameCommand),
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

func (e *mockPLogEvent) QName() schemas.QName { return e.Called().Get(0).(schemas.QName) }
func (e *mockPLogEvent) ArgumentObject() istructs.IObject {
	return e.Called().Get(0).(istructs.IObject)
}

type mockObject struct {
	istructs.IObject
	mock.Mock
}

func (o *mockObject) QName() schemas.QName { return o.Called().Get(0).(schemas.QName) }
