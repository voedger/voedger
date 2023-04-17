/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/cmd/edger/internal/states"
)

func Test_BasicUsageSuperController(t *testing.T) {
	require := require.New(t)

	super := New(
		map[states.AttributeKind]MicroControllerFactory{
			states.DockerStackAttribute: MockMicroControllerFactory,
			states.CommandAttribute:     MockMicroControllerFactory,
			states.EdgerAttribute:       MockMicroControllerFactory,
		},
		SuperControllerParams{},
	)

	desired := states.MakeDesiredState()
	desired["id"] = states.DesiredAttribute{
		Kind:   states.DockerStackAttribute,
		Offset: 1,
		Value:  "1.0.0.1",
	}

	require.NotNil(super)

	ctx := context.Background()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for ctx.Err() == nil {
			achieved, err := super.AchieveState(ctx, desired)

			require.NoError(err)

			if achieved.Achieves(desired) {
				break
			}

			time.Sleep(time.Microsecond)
		}
		wg.Done()
	}()

	wg.Wait()
}

func Test_SuperController(t *testing.T) {
	require := require.New(t)

	require.True(true)
}

func TestSuperControllerParams_achievedStateFilePath(t *testing.T) {

	cwd := func() string {
		s, err := os.Getwd()
		if err != nil {
			t.FailNow()
		}
		return s
	}

	tests := []struct {
		name string
		pars SuperControllerParams
		want string
	}{
		{
			name: "basic usage",
			pars: SuperControllerParams{AchievedStateFile: "/tmp/test/test.json"},
			want: "/tmp/test/test.json",
		},
		{
			name: "if empty, then return `edger-state.json` in cwd",
			pars: SuperControllerParams{},
			want: filepath.Join(cwd(), DefaultAchievedStateFileName),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pars.achievedStateFilePath(); got != tt.want {
				t.Errorf("SuperControllerParams.achievedStateFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_superController_loadStoreState(t *testing.T) {
	require := require.New(t)

	testTime := time.Now()

	testState := func() states.ActualState {
		s := states.MakeActualState()
		for a := states.DockerStackAttribute; a < states.AttributeKindCount; a++ {
			s[a.String()] = states.ActualAttribute{
				Kind:       a,
				Offset:     states.AttrOffset(100 + a),
				TimeMs:     testTime.UnixMilli(),
				AttemptNum: 1,
				Status:     states.FinishedStatus,
				Info:       fmt.Sprintf("%v success achieved", a),
			}
		}
		return s
	}

	t.Run("basic usage", func(t *testing.T) {
		super := superController{
			currentState: testState(),
			params:       SuperControllerParams{},
		}

		fn := super.params.achievedStateFilePath()
		defer func() { os.Remove(fn) }()

		err := super.storeState()
		require.NoError(err, err)

		err = super.loadState()
		require.NoError(err, err)

		require.Equal(testState(), super.currentState)
	})
}
