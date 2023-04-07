/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

type Projector struct {
	Name QName
	Func func(event IPLogEvent, state IState, intents IIntents) (err error)

	// When true, actualier doesn't buffer intents and apply them immediately after every event fed
	NonBuffered bool

	// If specified, the actualizer will only feed the declared events to istructs.Projector function. By default, all events fed.
	EventsFilter []QName

	// If specified, the actualizer will only feed the events with declared arguments to istructs.Projector function.
	// By default, events with any artuments fed.
	EventsArgsFilter []QName

	// If true, the actualizer also feds error events to istructs.Projector function. Default is false.
	HandleErrors bool
}

// ProjectorFactory creates a istructs.Projector
type ProjectorFactory func(partition PartitionID) Projector
