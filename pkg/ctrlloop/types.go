/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
* @author Alisher Nurmanov
 */

package ctrlloop

import "time"

type (
	ControllerFunction[Key comparable, SP any, State any, PV any] func(key Key, sp SP, state State) (newState *State, pv *PV, startTime *time.Time)
	ReporterFunction[Key comparable, PV any]                      func(key Key, pv *PV) (err error)
	nowTimeFunction                                               func() time.Time
)
