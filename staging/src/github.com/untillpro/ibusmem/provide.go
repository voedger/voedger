/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package ibusmem

import (
	"context"
	"time"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

// requestCtx is already contained by sender but exposed also as a separate param because it is more useful in request handlers
func Provide(requestHandler func(requestCtx context.Context, sender interface{}, request ibus.Request)) ibus.IBus {
	return provide(requestHandler, time.After, time.After, time.After)
}

func provide(requestHandler func(requestCtx context.Context, sender interface{}, request ibus.Request),
	timerResponse func(time.Duration) <-chan time.Time,
	timerSection func(time.Duration) <-chan time.Time,
	timerElement func(time.Duration) <-chan time.Time,
) ibus.IBus {
	if requestHandler == nil {
		panic("request handler must be not nil")
	}
	return &bus{
		requestHandler: requestHandler,
		timerResponse:  timerResponse,
		timerSection:   timerSection,
		timerElement:   timerElement,
	}
}
