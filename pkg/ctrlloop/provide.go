/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"sync"
)

var nextStartTimeFunc = getNextStartTime

// nolint
// New runs a control loop and returns func waiting for closing the loop
func New[Key comparable, SP any, PV any, State any](
	controllerFunc ControllerFunction[Key, SP, State, PV],
	reporterFunc ReporterFunction[Key, PV],
	numControllerRoutines int,
	ch chan ControlMessage[Key, SP],
	nowTimeFunc nowTimeFunction,
) (wait func()) {
	InProcess := sync.Map{}
	dedupInCh := make(chan statefulMessage[Key, SP, State])
	dedupOutCh := make(chan answer[Key, SP, PV, State])
	callerCh := make(chan statefulMessage[Key, SP, State])
	repeatCh := make(chan scheduledMessage[Key, SP, State], 3) // 3: scheduler, repeater, dedupIn
	repeaterCh := make(chan answer[Key, SP, PV, State])
	reporterCh := make(chan reportInfo[Key, PV])
	finishCh := make(chan struct{})

	go scheduler(ch, dedupInCh, repeatCh, nowTimeFunc)

	go dedupIn(dedupInCh, callerCh, repeatCh, &InProcess, nowTimeFunc)

	go dedupOut(dedupOutCh, repeaterCh, &InProcess)

	callerFinalizerCh := make(chan struct{}, numControllerRoutines-1)
	for i := 0; i < numControllerRoutines-1; i++ {
		callerFinalizerCh <- struct{}{}
	}

	close(callerFinalizerCh)

	for i := 0; i < numControllerRoutines; i++ {
		go caller(callerCh, dedupOutCh, callerFinalizerCh, controllerFunc)
	}

	go repeater(repeaterCh, repeatCh, reporterCh)

	go reporter(reporterCh, finishCh, reporterFunc)

	return func() {
		<-finishCh
	}
}
