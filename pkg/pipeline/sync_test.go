// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func opName(_ context.Context, work IWorkpiece) (err error) {
	work.(testwork).slots["name"] = "michael"
	return nil
}

func opAge(_ context.Context, work IWorkpiece) (err error) {
	work.(testwork).slots["age"] = "39"
	return nil
}

func opSex(_ context.Context, work IWorkpiece) (err error) {
	work.(testwork).slots["sex"] = "male"
	return nil
}

func opError(context.Context, IWorkpiece) (err error) {
	return errors.New("test failure")
}

func TestSyncPipeline_NotAWorkpiece(t *testing.T) {
	ctx := &testContext{}
	v := &notAWorkpiece{}
	pipeline := NewSyncPipeline(ctx, "my-pipeline", WireSyncOperator("noop", &NOOP{}))
	require.NoError(t, pipeline.DoSync(ctx, v))
}
