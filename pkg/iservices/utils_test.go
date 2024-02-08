/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_BasicUsage_WiredServicesToMap(t *testing.T) {
	require := require.New(t)

	var svc1 IMyService1 = &myService{dummy: 1}
	var svc2 IMyService2 = &myService{dummy: 2}
	var notSvc INotService = &NotServiceType{dummy: 3}

	ws := &WiredServices{MyService1: svc1, MyService2: svc2, NotService: notSvc}

	servicesMap := WiredStructPtrToMap(ws)

	require.Len(servicesMap, 2)
	require.Equal(1, servicesMap["MyService1"].(*myService).dummy)
	require.Equal(2, servicesMap["MyService2"].(*myService).dummy)

}

type IMyService1 interface {
	IService
}

type IMyService2 interface {
	IService
}

type myService struct{ dummy int }

func (ms *myService) Prepare() error          { return nil }
func (ms *myService) Run(ctx context.Context) {}

type INotService interface {
	Start() error
}

type NotServiceType struct{ dummy int }

func (ns *NotServiceType) Start() error { return nil }

type WiredServices struct {
	MyService1 IMyService1
	MyService2 IMyService2
	NotService INotService
}
