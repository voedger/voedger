/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

import (
	"fmt"
)

func puller_sync(wo *WiredOperator) {
	for work := range wo.Stdin {
		if work == nil {
			pipelinePanic("nil in puller_sync stdin", wo.name, wo.wctx)
		}
		if err, ok := work.(IErrorPipeline); ok {
			if catch, ok := wo.Operator.(ICatch); ok {
				if newerr := catch.OnErr(err, err.GetWork(), wo.wctx); newerr != nil {
					wo.Stdout <- wo.NewError(fmt.Errorf("nested error '%w' while handling '%s'", newerr, err.Error()), err.GetWork(), placeCatchOnErr)
					continue
				}
			} else {
				wo.Stdout <- err
				continue
			}
			work = err.GetWork() // restore from error
		}

		err := wo.doSync(wo.ctx, work)

		if err != nil {
			wo.Stdout <- err
		} else {
			wo.Stdout <- work
		}
	}
	wo.Operator.Close()
	close(wo.Stdout)
}
