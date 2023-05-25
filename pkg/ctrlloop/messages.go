/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"fmt"
	"time"
)

type ControlMessage[Key comparable, SP any] struct {
	Key                Key
	SP                 SP
	CronSchedule       string
	StartTimeTolerance time.Duration
}

func (m ControlMessage[Key, SP]) String() string {
	return fmt.Sprintf(keyLogFormat, m.Key)
}

type scheduledMessage[Key comparable, SP any, State any] struct {
	Key          Key
	SP           SP
	serialNumber uint64
	State        *State
	StartTime    time.Time
}

func (m scheduledMessage[Key, SP, State]) String() string {
	return fmt.Sprintf(keySerialNumberLogFormat, m.Key, m.serialNumber)
}

type answer[Key comparable, SP any, PV any, State any] struct {
	Key          Key
	SP           SP
	serialNumber uint64
	State        *State
	PV           *PV
	StartTime    *time.Time
}

func (m answer[Key, SP, PV, State]) String() string {
	return fmt.Sprintf(keySerialNumberLogFormat, m.Key, m.serialNumber)
}

type statefulMessage[Key comparable, SP any, State any] struct {
	Key          Key
	SP           SP
	serialNumber uint64
	State        State
}

func (m statefulMessage[Key, SP, State]) String() string {
	return fmt.Sprintf(keySerialNumberLogFormat, m.Key, m.serialNumber)
}

type reportInfo[Key comparable, PV any] struct {
	Key Key
	PV  *PV
}

func (m reportInfo[Key, PV]) String() string {
	return fmt.Sprintf(keyLogFormat, m.Key)
}

type reportInfoAttempt[Key comparable, PV any] struct {
	Key     Key
	PV      *PV
	Attempt int
}

func (m reportInfoAttempt[Key, PV]) String() string {
	return fmt.Sprintf(keyLogFormat, m.Key)
}
