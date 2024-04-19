/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservicesctl

import (
	"context"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iservices"
)

type servicesController struct {
	numOfServices int
}

func (sc *servicesController) PrepareAndRun(ctx context.Context, services map[string]iservices.IService) (join func(ctx context.Context), err error) {

	sc.numOfServices += len(services)

	prepared := make(map[string]iservices.IService)
	failed := make(map[string]error)

	// start services
	{
		wg := sync.WaitGroup{}
		mu := sync.Mutex{}
		prepareService := func(serviceName string, service iservices.IService) {
			defer wg.Done()
			err := service.Prepare()
			mu.Lock()
			if err == nil {
				logger.Info("service prepared:", serviceName)
				prepared[serviceName] = service
			} else {
				logger.Error("unable to prepare service:", serviceName, "error:", err)
				failed[serviceName] = err
			}
			mu.Unlock()
		}
		for k, s := range services {
			wg.Add(1)
			go prepareService(k, s)
		}
		wg.Wait()
	}

	if len(failed) > 0 {
		failedErrorTexts := make(map[string]string)
		for k, e := range failed {
			failedErrorTexts[k] = e.Error()
		}
		err := fmt.Errorf("%w: %#v", iservices.ErrAtLeastOneServiceFailedToStart, failedErrorTexts)
		logger.Info(err)
		return nil, err
	}

	runningServices := &sync.WaitGroup{}
	for k, s := range prepared {
		runningServices.Add(1)
		go func(serviceName string, service iservices.IService) {
			defer runningServices.Done()
			logger.Info("running service...:", serviceName)
			service.Run(ctx)
			logger.Info("service stopped:", serviceName)
		}(k, s)

	}

	join = func(ctx context.Context) { joinServices(ctx, runningServices) }
	return join, nil
}

//go:noinline
func joinServices(ctx context.Context, runningServices *sync.WaitGroup) {

	logger.Info("waiting for ctx...")
	<-ctx.Done()

	logger.Info("waiting for services...")
	runningServices.Wait()

	logger.Info("all services stopped")

}
