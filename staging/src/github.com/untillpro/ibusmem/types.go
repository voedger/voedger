/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package ibusmem

import (
	"context"
	"time"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type bus struct {
	requestHandler func(requestCtx context.Context, sender ibus.ISender, request ibus.Request)
	timerResponse  func(d time.Duration) <-chan time.Time
	timerSection   func(d time.Duration) <-chan time.Time
	timerElement   func(d time.Duration) <-chan time.Time
}

type channelSender struct {
	c         chan interface{}
	timeout   time.Duration
	clientCtx context.Context
}

type resultSenderClosable struct {
	currentSection ibus.ISection
	sections       chan ibus.ISection
	elements       chan element
	err            *error
	timeout        time.Duration
	clientCtx      context.Context // closed if client is e.g. disconnected
	timerSection   func(d time.Duration) <-chan time.Time
	timerElement   func(d time.Duration) <-chan time.Time
}

type arraySection struct {
	sectionType string
	path        []string
	elems       chan element
}

type mapSection struct {
	sectionType string
	path        []string
	elems       chan element
}

type objectSection struct {
	sectionType     string
	path            []string
	elements        chan element
	elementReceived bool
}

type element struct {
	name  string
	value []byte
}

type implISender struct {
	bus    ibus.IBus
	sender interface{}
}
