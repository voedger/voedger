/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservicesctl

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iservices"
)

func Test_BasicUsage(t *testing.T) {
	require := require.New(t)

	intf := New()

	services := make(map[string]iservices.IService)
	services["service1"] = &svcMock{name: "service1"}
	services["service2"] = &svcMock{name: "service2"}
	services["service3"] = &svcMock{name: "service2"}

	ctx, cancel := context.WithCancel(context.Background())

	join, err := intf.PrepareAndRun(ctx, services)
	require.NoError(err)

	cancel()
	join(ctx)

}

func Test_BasicUsage_Failure(t *testing.T) {
	require := require.New(t)

	intf := New()

	services := make(map[string]iservices.IService)
	services["service1"] = &svcMock{name: "service1"}
	services["service2"] = &svcMock{name: "service2", prepareShouldFail: true}
	services["service3"] = &svcMock{name: "service3"}
	services["service4"] = &svcMock{name: "service4", prepareShouldFail: true}

	join, err := intf.PrepareAndRun(context.Background(), services)
	require.Error(err)
	require.Nil(join)

	require.False(services["service1"].(*svcMock).runCalled)
	require.False(services["service3"].(*svcMock).runCalled)
	require.False(services["service2"].(*svcMock).runCalled)
	require.False(services["service4"].(*svcMock).runCalled)
}

type svcMock struct {
	prepareShouldFail bool
	prepareCalled     bool
	runCalled         bool
	name              string
}

func (svc *svcMock) Prepare() (err error) {
	svc.prepareCalled = true
	if svc.prepareShouldFail {
		return fmt.Errorf("error %v", svc.name)
	}
	return nil
}

func (svc *svcMock) Run(ctx context.Context) {
	svc.runCalled = true
	<-ctx.Done()
}

func Test_StartStop(t *testing.T) {
	require := require.New(t)

	intf := New()

	services := make(map[string]iservices.IService)
	services["service1"] = &svcMock{name: "service1"}
	services["service2"] = &svcMock{name: "service2"}
	services["service3"] = &svcMock{name: "service3"}
	services["service4"] = &svcMock{name: "service4"}

	ctx, cancel := context.WithCancel(context.Background())
	join, err := intf.PrepareAndRun(ctx, services)
	require.NoError(err)
	require.NotNil(join)

	require.True(services["service1"].(*svcMock).prepareCalled)
	require.True(services["service2"].(*svcMock).prepareCalled)
	require.True(services["service3"].(*svcMock).prepareCalled)
	require.True(services["service4"].(*svcMock).prepareCalled)

	cancel()
	join(ctx)

	require.True(services["service1"].(*svcMock).runCalled)
	require.True(services["service2"].(*svcMock).runCalled)
	require.True(services["service3"].(*svcMock).runCalled)
	require.True(services["service4"].(*svcMock).runCalled)
}

func Test_fixLinterUnused(t *testing.T) {
	require := require.New(t)

	sc := &servicesController{}
	require.Equal(0, sc.numOfServices)
}
