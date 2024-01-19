/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package ibusmem

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	t.Run("Single response basic usage", func(t *testing.T) {
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {

			sender.SendResponse(ibus.Response{Data: []byte("hello world")})

			// second response -> panic
			require.Panics(func() { sender.SendParallelResponse() })
			require.Panics(func() { sender.SendResponse(ibus.Response{}) })

		})

		response, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)
		require.Equal("hello world", string(response.Data))
		require.Nil(sections)
		require.Nil(secErr)
		require.Nil(err)
	})
	t.Run("Sectioned response basic usage", func(t *testing.T) {
		testErr := errors.New("error from result sender")
		ch := make(chan interface{})
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {

			rs := sender.SendParallelResponse()
			go func() {

				// try to send an element without section -> panic
				require.Panics(func() { require.Nil(rs.SendElement("", "element1")) })

				rs.StartArraySection("array", []string{"array-path"})
				require.Nil(rs.SendElement("", "element1"))
				require.Nil(rs.SendElement("", "element2"))
				require.Nil(rs.SendElement("", nil)) // nothingness will not be sent

				// send an unmarshalable element -> error
				require.NotNil(rs.SendElement("", func() {}))

				require.Nil(rs.ObjectSection("object", []string{"object-path"}, "value"))
				require.Nil(rs.ObjectSection("", nil, nil)) // nothingness will not be sent

				// try to send element on object section -> panic
				require.Panics(func() { rs.SendElement("", 42) })

				rs.StartMapSection("map", []string{"map-path"})
				require.Nil(rs.SendElement("key1", "value1"))
				require.Nil(rs.SendElement("key2", "value2"))
				require.Nil(rs.SendElement("", nil)) // nothingness will not be sent

				rs.Close(testErr)

				// any action after close -> panic
				require.Panics(func() { rs.SendElement("", 42) })
				require.Panics(func() { rs.StartArraySection("array", []string{"array-path"}) })
				require.Panics(func() { rs.StartMapSection("array", []string{"array-path"}) })
				require.Panics(func() { rs.ObjectSection("array", []string{"array-path"}, nil) })
				close(ch)
			}()

			// second response -> panic
			require.Panics(func() { sender.SendParallelResponse() })
			require.Panics(func() { sender.SendResponse(ibus.Response{}) })
		})

		requestCtx := context.Background()
		response, sections, secErr, err := bus.SendRequest2(requestCtx, ibus.Request{}, ibus.DefaultTimeout)
		require.Nil(err)
		require.Empty(response)

		// expect array section
		section := <-sections
		arraySection := section.(ibus.IArraySection)
		require.Equal("array", arraySection.Type())
		require.Equal([]string{"array-path"}, arraySection.Path())

		// elems of array section
		val, ok := arraySection.Next(requestCtx)
		require.True(ok)
		require.Equal(`"element1"`, string(val))
		val, ok = arraySection.Next(requestCtx)
		require.True(ok)
		require.Equal(`"element2"`, string(val))
		val, ok = arraySection.Next(requestCtx)
		require.Nil(val)
		require.False(ok)

		// expect object section
		section = <-sections
		objectSection := section.(ibus.IObjectSection)
		require.Equal("object", objectSection.Type())
		require.Equal([]string{"object-path"}, objectSection.Path())
		require.Equal(`"value"`, string(objectSection.Value(requestCtx)))

		// expect map section
		section = <-sections
		mapSection := section.(ibus.IMapSection)
		require.Equal("map", mapSection.Type())
		require.Equal([]string{"map-path"}, mapSection.Path())

		// elems of map section
		name, val, ok := mapSection.Next(requestCtx)
		require.True(ok)
		require.Equal("key1", name)
		require.Equal(`"value1"`, string(val))
		name, val, ok = mapSection.Next(requestCtx)
		require.True(ok)
		require.Equal("key2", name)
		require.Equal(`"value2"`, string(val))
		name, val, ok = mapSection.Next(requestCtx)
		require.False(ok)
		require.Equal("", name)
		require.Nil(val)

		// no more sections
		_, ok = <-sections
		require.False(ok)
		require.ErrorIs(testErr, *secErr)
		<-ch
	})
	t.Run("Provide should panic on nil requestHandler provided", func(t *testing.T) {
		require.Panics(func() { Provide(nil) })
	})
	t.Run("Should return timeout error if no response at all", func(t *testing.T) {
		bus := provide(func(srequestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			// do not send response to trigger the timeout case
		}, timeoutTrigger, time.After, time.After)

		response, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)

		require.Equal(ibus.ErrTimeoutExpired, err)
		require.Empty(response)
		require.Nil(sections)
		require.Nil(secErr)
	})
}

func TestBus_SendRequest2(t *testing.T) {
	require := require.New(t)
	t.Run("Should return error on ctx done", func(t *testing.T) {
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			// do not send response to trigger ctx.Done() case
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		response, sections, secErr, err := bus.SendRequest2(ctx, ibus.Request{}, ibus.DefaultTimeout)

		require.Equal("context canceled", err.Error())
		require.Empty(response)
		require.Nil(sections)
		require.Nil(secErr)
	})
	t.Run("Should handle panic in request handler with message", func(t *testing.T) {
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			panic("boom")
		})

		response, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)

		require.Equal("boom", err.Error())
		require.Empty(response)
		require.Nil(sections)
		require.Nil(secErr)
	})
	t.Run("Should handle panic in request handler with error", func(t *testing.T) {
		testErr := errors.New("boom")
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			panic(testErr)
		})

		response, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)

		require.ErrorIs(err, testErr)
		require.Empty(response)
		require.Nil(sections)
		require.Nil(secErr)
	})
}

func TestResultSenderClosable_StartArraySection(t *testing.T) {
	require := require.New(t)
	t.Run("Should return timeout error on long section read", func(t *testing.T) {
		ch := make(chan interface{})
		bus := provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				err := rs.ObjectSection("", nil, 42)
				ch <- nil // signal tried to send a section
				require.ErrorIs(err, ibus.ErrNoConsumer)
				rs.Close(nil)
			}()
		}, time.After, timeoutTrigger, time.After)
		resp, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)
		require.Nil(err)
		require.NotNil(sections)
		require.Empty(resp)
		<-ch // do not read section to trigger the timeout case
		_, ok := <-sections
		require.False(ok)
		require.Nil(*secErr)

	})
	t.Run("Should return error when ctx done on send section", func(t *testing.T) {
		ch := make(chan interface{})
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				<-ch // wait for cancel
				err := rs.ObjectSection("", nil, 42)
				ch <- nil // signal ok to start read sections. That forces ctx.Done case fire at tryToSendSection
				require.Equal("context canceled", err.Error())
				rs.Close(nil)
			}()
		})
		ctx, cancel := context.WithCancel(context.Background())
		response, sections, secErr, err := bus.SendRequest2(ctx, ibus.Request{}, ibus.DefaultTimeout)
		require.Nil(err)
		require.Empty(response)
		require.NotNil(sections)
		cancel()
		ch <- nil // signal cancelled
		<-ch      // delay to read sections to make ctx.Done() branch at tryToSendSection fire.
		// note: section could be sent on ctx.Done() because cases order is undefined at tryToSendSection. But ObjectSection() will return error in any case
		for range sections {
		}
		require.Nil(*secErr)
	})
}

func TestResultSenderClosable_SendElement(t *testing.T) {
	require := require.New(t)
	t.Run("Should accept object", func(t *testing.T) {
		type article struct {
			ID   int64
			Name string
		}
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				require.Nil(rs.ObjectSection("", nil, article{ID: 100, Name: "Cola"}))
				rs.Close(nil)
			}()
		})
		a := article{}

		requestCtx := context.Background()
		response, sections, secErr, err := bus.SendRequest2(requestCtx, ibus.Request{}, ibus.DefaultTimeout)

		require.Nil(err)
		require.Empty(response)
		require.Nil(json.Unmarshal((<-sections).(ibus.IObjectSection).Value(requestCtx), &a))
		require.Equal(int64(100), a.ID)
		require.Equal("Cola", a.Name)
		_, ok := <-sections
		require.False(ok)
		require.Nil(*secErr)
	})
	t.Run("Should accept JSON", func(t *testing.T) {
		type point struct {
			X int64
			Y int64
		}
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				require.Nil(rs.ObjectSection("", nil, []byte(`{"X":52,"Y":89}`)))
				rs.Close(nil)
			}()
		})
		p := point{}

		requestCtx := context.Background()
		response, sections, secErr, err := bus.SendRequest2(requestCtx, ibus.Request{}, ibus.DefaultTimeout)
		require.Nil(err)
		require.Empty(response)
		require.Nil(json.Unmarshal((<-sections).(ibus.IObjectSection).Value(requestCtx), &p))
		require.Equal(int64(52), p.X)
		require.Equal(int64(89), p.Y)
		_, ok := <-sections
		require.False(ok)
		require.Nil(*secErr)
	})
	t.Run("Should return error when client reads element too long", func(t *testing.T) {
		bus := provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				rs.StartArraySection("", nil)
				require.ErrorIs(rs.SendElement("", 0), ibus.ErrNoConsumer)
				rs.Close(nil)
			}()
		}, time.After, time.After, timeoutTrigger)

		response, sections, secErr, err := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)

		require.Nil(err)
		require.Empty(response)
		_ = (<-sections).(ibus.IArraySection)
		// do not read an element to trigger timeout
		_, ok := <-sections
		require.False(ok)
		require.Nil(*secErr)
	})
	t.Run("Should return error when ctx done on send element", func(t *testing.T) {
		ch := make(chan interface{})
		bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
			rs := sender.SendParallelResponse()
			go func() {
				rs.StartArraySection("", nil)
				require.Nil(rs.SendElement("", 0))
				<-ch // wait for element read
				<-ch // wait for cancel
				err := rs.SendElement("", 1)
				ch <- nil // signal ok to read next element. That forces ctx.Done() case fire at tryToSendElement

				require.Equal("context canceled", err.Error())
				rs.Close(nil)
			}()
		})
		ctx, cancel := context.WithCancel(context.Background())
		response, sections, secErr, err := bus.SendRequest2(ctx, ibus.Request{}, ibus.DefaultTimeout)
		require.Nil(err)
		require.Empty(response)
		array := (<-sections).(ibus.IArraySection)
		val, ok := array.Next(ctx)
		require.True(ok)
		require.Equal([]byte("0"), val)
		ch <- nil // signal we're read an element
		// cancel should be synchronzied, otherwise possible at tryToSendElement: write to elements channel, read it here, cancel here,
		// then return non-nil ctx.Err() from tryToSendElement -> SendElement early failed
		cancel()
		ch <- nil              // signal cancelled
		<-ch                   // wait for ok to read next element to force ctx.Done case fire at tryToSendElement
		_, _ = array.Next(ctx) // note: element could be sent on ctx.Done() because cases order is undefined at tryToSendElement. But SendElement() will return error in any case
		_, ok = <-sections
		require.False(ok)
		require.Nil(*secErr)
	})
}

func TestObjectSection_Value(t *testing.T) {
	require := require.New(t)
	bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		rs := sender.SendParallelResponse()
		go func() {
			require.Nil(rs.ObjectSection("", nil, []byte("bb")))
			rs.Close(nil)
		}()
	})

	requestCtx := context.Background()
	response, sections, secErr, err := bus.SendRequest2(requestCtx, ibus.Request{}, ibus.DefaultTimeout)

	object := (<-sections).(ibus.IObjectSection)
	require.Nil(err)
	require.Empty(response)
	require.Equal([]byte("bb"), object.Value(requestCtx))
	require.Nil(object.Value(requestCtx))

	_, ok := <-sections
	require.False(ok)
	require.Nil(*secErr)
}

func TestClientDisconnectOnSectionsSending(t *testing.T) {
	require := require.New(t)
	ch := make(chan interface{})
	bus := Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		rs := sender.SendParallelResponse()
		go func() {
			rs.StartArraySection("array", []string{"array-path"})
			require.Nil(rs.SendElement("1", "element1"))
			ch <- nil
			// client is disconnected here
			<-ch
			require.Error(context.Canceled, rs.SendElement("2", "element2"))
			rs.Close(nil)
			close(ch)
		}()
	})

	clientCtx, clientCtxCancel := context.WithCancel(context.Background())
	response, sections, secErr, err := bus.SendRequest2(clientCtx, ibus.Request{}, ibus.DefaultTimeout)
	require.Nil(err)
	require.Empty(response)

	// expect array section
	section := <-sections
	arraySection := section.(ibus.IArraySection)
	require.Equal("array", arraySection.Type())
	require.Equal([]string{"array-path"}, arraySection.Path())

	// elems of array section
	val, ok := arraySection.Next(clientCtx)
	require.True(ok)
	require.Equal(`"element1"`, string(val))

	// simulate client disconnect
	<-ch
	clientCtxCancel()
	ch <- nil

	val, ok = arraySection.Next(clientCtx)
	if ok {
		// client context canceled -> write to sections channel is possible anyway because
		// sectons<- and <-ctx.Done() cases has the same probability at resultSenderClosable.tryToSendElement(). ctx.Err() will be retuned even on sectons<-
		require.Equal(`"element2"`, string(val))
		_, ok = arraySection.Next(clientCtx)
		require.False(ok)
	}

	// no more sections
	_, ok = <-sections
	require.False(ok)
	require.Nil(*secErr)
	<-ch
}

func timeoutTrigger(d time.Duration) <-chan time.Time {
	res := make(chan time.Time)
	close(res)
	return res
}
