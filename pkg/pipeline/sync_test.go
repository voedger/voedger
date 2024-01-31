/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func opName(_ context.Context, work interface{}) (err error) {
	work.(testwork).slots["name"] = "michael"
	return nil
}

func opAge(_ context.Context, work interface{}) (err error) {
	work.(testwork).slots["age"] = "39"
	return nil
}

func opSex(_ context.Context, work interface{}) (err error) {
	work.(testwork).slots["sex"] = "male"
	return nil
}

func opError(context.Context, interface{}) (err error) {
	return errors.New("test failure")
}

func TestSyncPipeline_NotAWorkpiece(t *testing.T) {
	type notAWorkpiece struct{}
	ctx := &testContext{}
	v := &notAWorkpiece{}
	pipeline := NewSyncPipeline(ctx, "my-pipeline", WireSyncOperator("noop", &NOOP{}))
	require.Nil(t, pipeline.DoSync(ctx, v))
}
