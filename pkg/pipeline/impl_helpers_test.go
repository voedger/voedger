// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko 



package pipeline

import (
	"context"
	"time"
)

type funcOnErr func(err error, work interface{}, context IWorkpieceContext) (newErr error)

type testContext struct {
	done chan struct{}
	err  error
}

func (c testContext) Done() <-chan struct{} {
	return c.done
}

func (c testContext) Err() error {
	return c.err
}

func (c testContext) Deadline() (deadline time.Time, ok bool) {
	return time.Now(), false
}

func (c testContext) Value(key interface{}) interface{} {
	return nil
}

type testwork struct {
	slots map[string]interface{}
}

func (w testwork) Release() {}

func newTestWork() testwork {
	return testwork{
		slots: make(map[string]interface{}),
	}
}

type mockIService struct {
	run     func(ctx context.Context)
	stop    func()
	prepare func(work interface{}) error
}

func (s *mockIService) Run(ctx context.Context) {
	s.run(ctx)
}

func (s *mockIService) Stop() {
	s.stop()
}

func (s *mockIService) Prepare(work interface{}) error {
	return s.prepare(work)
}

type testWorkpiece struct {
	release func()
}

func (w testWorkpiece) Release() {
	w.release()
}
