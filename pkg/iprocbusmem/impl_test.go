/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iprocbusmem

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	pb := Provide([]ChannelGroup{
		{
			NumChannels:       10,
			ChannelBufferSize: 1,
		},
		{
			NumChannels:       1,
			ChannelBufferSize: 0,
		},
	})

	// chan buffer 1 -> no reader required
	require.True(pb.Submit(0, 3, 42))
	mes := <-pb.ServiceChannel(0, 3)
	require.Equal(42, mes)

	// no reader -> false
	require.False(pb.Submit(1, 0, 43))

	done := make(chan interface{})
	// start reader
	go func() {
		mes := <-pb.ServiceChannel(1, 0)
		require.Equal(44, mes)
		done <- nil
	}()
	// submit until somebody read
	for !pb.Submit(1, 0, 44) {
	}
	<-done
}

func TestErrors(t *testing.T) {
	require := require.New(t)
	pb := Provide([]ChannelGroup{
		{
			NumChannels:       10,
			ChannelBufferSize: 1,
		},
		{
			NumChannels:       1,
			ChannelBufferSize: 0,
		},
	})

	require.Panics(func() { pb.ServiceChannel(2, 2) }) // wrong groupIdx
	require.Panics(func() { pb.ServiceChannel(1, 2) }) // wrong channelIdx
	require.Panics(func() { pb.Submit(2, 2, nil) })    // wrong groupIdx
	require.Panics(func() { pb.Submit(1, 2, nil) })    // wrong channelIdx
}
