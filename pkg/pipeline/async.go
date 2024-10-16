// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"time"
)

func puller_async(wo *WiredOperator) {
	timer := time.NewTimer(wo.FlushInterval)
	timer.Stop()
	var open = true
	var work interface{}
	timerTicked := true
	for open {
		select {
		case work, open = <-wo.Stdin:

			if !open {
				continue
			}

			workpiece := work.(IWorkpiece)

			if !wo.isActive() {
				p_release(workpiece)
				continue
			}

			if wo.forwardIfErrorAsync(workpiece) {
				continue
			}

			outWork, err := wo.doAsync(workpiece)

			if err != nil {
				wo.Stdout <- err
			} else {
				if outWork != nil {
					wo.Stdout <- outWork
				}
				if timerTicked && wo.FlushInterval > 0 {
					timer.Reset(wo.FlushInterval)
					timerTicked = false
				}
			}
		case <-timer.C:
			timerTicked = true
			p_flush(wo, placeFlushByTimer)
		}
	}

	p_flush(wo, placeFlushDisassembling)
	wo.Operator.Close()
	close(wo.Stdout)
}

func p_flush(wo *WiredOperator, place string) {
	if !wo.isActive() {
		return
	}

	if err := wo.Operator.(IAsyncOperator).Flush(wo.flushCB); err != nil {
		if wo.isActive() {
			wo.Stdout <- wo.NewError(err, nil, place)
		}
	}
}

func p_release(w IWorkpiece) {
	if w != nil {
		w.Release()
	}
}
