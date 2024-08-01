// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwitchOperator_Close(t *testing.T) {
	closed := false
	operator := switchOperator{
		branches: map[string]ISyncOperator{"branch": mockSyncOp().
			close(func() {
				closed = true
			}).create()},
	}

	operator.Close()

	require.True(t, closed)
}

func TestSwitchOperator_DoSync(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		doSync := false
		operator := switchOperator{
			branches: map[string]ISyncOperator{
				"branch": NewSyncOp(func(ctx context.Context, work IWorkpiece) (err error) {
					doSync = true
					return nil
				}),
			},
			switchLogic: mockSwitch{func(work interface{}) (branch string, err error) {
				return "branch", nil
			}},
		}

		_ = operator.DoSync(context.Background(), nil)

		require.True(t, doSync)
	})
	t.Run("Should be not ok because switch logic return error", func(t *testing.T) {
		operator := switchOperator{
			switchLogic: mockSwitch{func(work interface{}) (branch string, err error) {
				return "branch", errors.New("switch logic error")
			}},
		}

		err := operator.DoSync(context.Background(), nil)

		require.Equal(t, "switch logic error", err.Error())
	})
}

func TestSwitchOperator(t *testing.T) {
	t.Run("Should create switch operator", func(t *testing.T) {
		switchOperator := SwitchOperator(mockSwitch{}, SwitchBranch("branch", mockSyncOp().create()))

		require.NotNil(t, switchOperator)
	})

	t.Run("Should panic when switch logic is nil", func(t *testing.T) {
		require.PanicsWithValue(t, "switch must be not nil", func() {
			SwitchOperator(nil, SwitchBranch("branch", mockSyncOp().create()))
		})
	})
}

type mockSwitch struct {
	switchLogic func(work interface{}) (branchName string, err error)
}

func (s mockSwitch) Switch(work interface{}) (branchName string, err error) {
	return s.switchLogic(work)
}
