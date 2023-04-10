/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservices

import "context"

type IService interface {
	Prepare() (err error)
	Run(ctx context.Context)
}

// IServiceFactory is a factory for IService
type IServicesController interface {
	// PrepareAndRun services (IService) in separate goroutines
	// If any service fails then
	//   Run() is called with cancelled context for every started service
	//   errors.Is(ErrAtLeastOneServiceFailedToStart, err)
	// If all servies are ok join() should be called to join services
	// join() waits ctx and then waits for all services
	PrepareAndRun(ctx context.Context, services map[string]IService) (join func(ctx context.Context), err error)
}
