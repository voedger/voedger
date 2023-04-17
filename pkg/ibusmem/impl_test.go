/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibusmem

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/ibus"
)

func Test_BasicUsage_RegisterReceivers_QuerySender(t *testing.T) {
	require := require.New(t)
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	// EchoReceiver, two reading goroutines, channel buffer is 10
	busimpl.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 2, 10)

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	response, _, err := sender.Send(context.Background(), "hello123", ibus.NullHandler)
	require.Nil(err)

	require.Equal("hello123", response)

	// NOTE: If missed will be done by cleanup
	busimpl.UnregisterReceiver("owner", "app", 0, "q")
}

func responseWithSections(_ context.Context, request interface{}, sectionsWriter ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
	numSections := request.(int)
	for i := 0; i < numSections; i++ {
		sectionsWriter.Write(i)
	}
	return ibus.NewResult(numSections, nil, "", "")
}

func Test_BasicUsage_Sections(t *testing.T) {
	require := require.New(t)
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	busimpl.RegisterReceiver("owner", "app", 0, "q", responseWithSections, 2, 10)

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	for requestedNumSections := 0; requestedNumSections < 50; requestedNumSections++ {
		sections := make([]interface{}, 0)
		sectionsWriter := func(responsePart interface{}) {
			sections = append(sections, responsePart)
		}

		response, _, err := sender.Send(context.Background(), requestedNumSections, sectionsWriter)

		require.Nil(err)
		require.Equal(requestedNumSections, response)
		require.Equal(requestedNumSections, len(sections))
		for i := 0; i < requestedNumSections; i++ {
			require.Equal(i, sections[i])
		}
	}
}

func Test_BasicUsage_ErrClientClosedRequest(t *testing.T) {
	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	receiver := func(_ context.Context, msg interface{}, writer ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		for writer.Write(msg) {
		}
		return nil, ibus.Status{HTTPStatus: http.StatusOK}, nil
	}
	bus.RegisterReceiver("owner", "app", 0, "q", receiver, 1, 10)

	sender, ok := bus.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	ctx, cancel := context.WithTimeout(context.Background(), rwTimeout/2) // rwTimeout/2 caused the Bug_InitiallyClosedContext
	defer cancel()

	_, status, err := sender.Send(ctx, "msg", ibus.NullHandler)

	require.True(errors.Is(err, ibus.ErrClientClosedRequest), err)
	require.Equal(ibus.StatusClientClosedRequest, status.HTTPStatus)

}

func Test_BasicUsage_SlowReceiver(t *testing.T) {

	require := require.New(t)
	rwTimeout := time.Millisecond * 10
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	receiver := func(_ context.Context, msg interface{}, write ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		time.Sleep(rwTimeout * 4)
		return nil, ibus.Status{HTTPStatus: http.StatusOK}, nil
	}
	bus.RegisterReceiver("slowowner", "slowapp", 0, "q", receiver, 1, 10)

	sender, ok := bus.QuerySender("slowowner", "slowapp", 0, "q")
	require.True(ok)

	// NOTE: ibus.NullHandler skips all messages
	_, status, err := sender.Send(context.Background(), "msg", ibus.NullHandler)
	require.True(errors.Is(err, ibus.ErrReadTimeoutExpired), err)
	require.Equal(http.StatusGatewayTimeout, status.HTTPStatus)
	require.True(strings.Contains(status.ErrorMessage, "slowowner"))
	require.True(strings.Contains(status.ErrorMessage, "slowapp"))
}

func Test_BasicUsage_SlowClient(t *testing.T) {

	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	// endless writing loop will be terminated by context since no one will read it
	receiver := func(_ context.Context, msg interface{}, writer ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		for writer.Write("hello") {
		}
		return nil, ibus.Status{HTTPStatus: http.StatusOK}, nil
	}
	bus.RegisterReceiver("owner", "app", 0, "q", receiver, 1, 10)

	sender, ok := bus.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	slowSectionsHandler := func(responsePart interface{}) {
		time.Sleep(rwTimeout * 4)
	}

	_, status, err := sender.Send(context.Background(), "msg", slowSectionsHandler)
	require.True(errors.Is(err, ibus.ErrSlowClient), err)
	require.Equal(http.StatusGatewayTimeout, status.HTTPStatus)
}

func Test_BasicUsage_QuerySender_ReceiverNotFound(t *testing.T) {

	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	// NOTE: receiver not registered but it is possible to use Sender
	sender, ok := bus.QuerySender("owner", "app", 0, "q")
	require.False(ok)

	_, status, err := sender.Send(context.Background(), "msg", ibus.NullHandler)
	require.True(errors.Is(err, ibus.ErrReceiverNotFound), err)
	require.True(strings.Contains(status.ErrorMessage, "owner"))
	require.Equal(http.StatusBadRequest, status.HTTPStatus)
}

/*
=== RUN   Test_ErrServiceUnavailable
06/21 15:48:10.164: ===: [ibusmem.(*bus).RegisterReceiver:264]: receiver registered: addr: owner/app/0/q, numOfProcessors: 1
06/21 15:48:10.164: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.164: ===: [ibusmem.Test_ErrServiceUnavailable.func1:180]: freezing... freeze0
06/21 15:48:10.179: ===: [ibusmem.Test_ErrServiceUnavailable.func1:180]: freezing... freeze1
06/21 15:48:10.179: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.179: ===: [ibusmem.Test_ErrServiceUnavailable.func2:205]: request 0 sent: freeze0
06/21 15:48:10.194: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.194: ===: [ibusmem.Test_ErrServiceUnavailable.func2:205]: request 1 sent: freeze1
06/21 15:48:10.209: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.224: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.240: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
06/21 15:48:10.255: ===: [ibusmem.Test_ErrServiceUnavailable:211]: waiting for two concurrent requests
*/

func Test_ErrServiceUnavailable(t *testing.T) {
	require := require.New(t)
	rwTimeout := time.Millisecond * 200
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()
	ctx := context.Background()

	receiver := func(_ context.Context, request interface{}, sectionsWriter ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		if strings.Contains(request.(string), "freeze") {
			logger.Info("freezing...", request)
			// will be two freezing in a row, so we should not exceed receiver timeout
			time.Sleep(rwTimeout / 4)
		}
		return nil, ibus.Status{HTTPStatus: http.StatusServiceUnavailable}, nil
	}
	busimpl.RegisterReceiver("owner", "app", 0, "q", receiver, 1, 1)

	var wg sync.WaitGroup
	// Run two goroutines to send messages to the bus, first freezes receiver, second drains AddressHandler channel
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
			require.True(ok)
			msg := fmt.Sprintf("freeze%d", i)
			for {
				_, _, err := sender.Send(ctx, msg, ibus.NullHandler)
				if errors.Is(err, ibus.ErrServiceUnavailable) {
					logger.Info("ServiceUnavailable happened:", msg)
					continue
				}
				break
			}
			logger.Info("request", i, "sent:", msg)
		}(i)
	}

	start := time.Now()
	for busimpl.GetMetrics().NumOfConcurrentRequests < 2 && time.Since(start) < rwTimeout*2 {
		logger.Info("waiting for two concurrent requests")
		time.Sleep(rwTimeout / 20)
	}
	require.True(time.Since(start) < rwTimeout*2, "time.Since(start) > rwTimeout*2")

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	require.True(ok)
	_, status, err := sender.Send(ctx, "hello", ibus.NullHandler)

	require.True(errors.Is(err, ibus.ErrServiceUnavailable), err)
	require.Equal(http.StatusServiceUnavailable, status.HTTPStatus)

	wg.Wait()
}

func Test_ErrBusUnavailable(t *testing.T) {

	require := require.New(t)

	rwTimeout := time.Second * 10
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 1, ReadWriteTimeout: rwTimeout})

	defer cleanup()
	ctx := context.Background()

	receiverChannel := make(chan struct{})
	receiver := func(_ context.Context, request interface{}, sectionsWriter ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		logger.Info("receiver is sleeping, request:", request)
		<-receiverChannel
		logger.Info("receiver is awake", request)
		return nil, ibus.Status{HTTPStatus: http.StatusServiceUnavailable}, nil
	}
	busimpl.RegisterReceiver("owner", "app", 0, "q", receiver, 1, 1)

	var wg sync.WaitGroup

	// goroutine freezes receiver and drains bus
	wg.Add(1)
	go func() {
		defer wg.Done()
		sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
		require.True(ok)
		_, _, err := sender.Send(ctx, 1, ibus.NullHandler)
		require.NoError(err)
		logger.Info("request", 1, "sent")
	}()

	for busimpl.GetMetrics().NumOfConcurrentRequests < 1 {
		logger.Info("waiting for one concurrent request")
		time.Sleep(time.Millisecond * 10)
	}
	logger.Info("one concurrent request - done")

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	require.True(ok)
	_, status, err := sender.Send(ctx, "hello", ibus.NullHandler)
	require.True(errors.Is(err, ibus.ErrBusUnavailable), err)
	require.Equal(http.StatusServiceUnavailable, status.HTTPStatus)

	receiverChannel <- struct{}{}
	wg.Wait()

}

func Test_ErrSlowClient_OnDifferentSections(t *testing.T) {

	// Similar to BasicUsage_SlowClient but client freezes after a given section number

	require := require.New(t)
	rwTimeout := time.Millisecond * 10
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	// endless writing loop will be terminated by context since no one will read it
	receiver := func(_ context.Context, msg interface{}, writer ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
		cnt := 0
		for writer.Write(fmt.Sprint("msg: ", msg, ", section:", cnt)) {
			cnt++
		}
		return msg, ibus.Status{HTTPStatus: http.StatusOK}, nil
	}
	bus.RegisterReceiver("owner", "app", 0, "q", receiver, 1, 10)

	sender, ok := bus.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	// range 10
	for i := 0; i < 10; i++ {
		sectionNumber := 0
		slowSectionsHandler := func(section interface{}) {
			if sectionNumber == i {
				logger.Info("freezing...")
				time.Sleep(rwTimeout * 4)
			}
			sectionNumber++
		}
		_, status, err := sender.Send(context.Background(), i, slowSectionsHandler)
		require.True(errors.Is(err, ibus.ErrSlowClient), err)
		require.Equal(http.StatusGatewayTimeout, status.HTTPStatus)
	}
}

func Test_RegisterReceivers_PanicIfReceiversAlreadyRegistered(t *testing.T) {

	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	bus.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 2, 10)
	require.Panics(func() {
		bus.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 2, 10)
	})

}

func Test_UnregisterReceivers_FalseIfReceiversNotRegistered(t *testing.T) {

	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	require.False(bus.UnregisterReceiver("owner", "app", 0, "q"))
}

// 06/20 18:49:03.201: ===: [ibusmem.(*bus).RegisterReceiver:247]: receiver registered: addr: owner/app/0/q, numOfProcessors: 2
// 06/20 18:49:04.206: !!!: [ibusmem.(*addressHandlerType).Send:151]: ibus.ErrReadTimeoutExpired addr: owner/app/0/q, numOfProcessors: 2 , request: 23
//
//	e:\workspaces\heeus\core\ibusmem\impl_test.go:69:
//	    	Error Trace:	impl_test.go:69
//	    	Error:      	Expected nil, but got: &errors.errorString{s:"ibus.ErrReadTimeoutExpired"}
//	    	Test:       	Test_BasicUsage_Sections
//
// Reason:  non-blocking write caused message to be missed, since sender was busy
func Test_Bug_ErrReadTimeoutExpired_Sections(t *testing.T) {

	require := require.New(t)
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	busimpl.RegisterReceiver("owner", "app", 0, "q", responseWithSections, 2, 10)

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	for i := 0; i < 20; i++ {
		const requestedNumSections = 23

		sections := make([]interface{}, 0)
		sectionsWriter := func(responsePart interface{}) {
			sections = append(sections, responsePart)
		}

		response, _, err := sender.Send(context.Background(), requestedNumSections, sectionsWriter)

		require.Nil(err)
		require.Equal(requestedNumSections, response)
		require.Equal(requestedNumSections, len(sections))
		for i := 0; i < requestedNumSections; i++ {
			require.Equal(i, sections[i])
		}
	}
}

// processor did not close responseChannel when senderContext.Err() != nil
func Test_Bug_InitiallyClosedContext(t *testing.T) {
	require := require.New(t)
	rwTimeout := time.Millisecond * 50
	bus, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: rwTimeout})
	defer cleanup()

	bus.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 1, 10)

	sender, ok := bus.QuerySender("owner", "app", 0, "q")
	require.True(ok)

	ctx, cancel := context.WithTimeout(context.Background(), rwTimeout/2)
	cancel()
	defer cancel()

	_, status, err := sender.Send(ctx, "msg", ibus.NullHandler)

	require.True(errors.Is(err, ibus.ErrClientClosedRequest), err)
	require.Equal(ibus.StatusClientClosedRequest, status.HTTPStatus)
}
