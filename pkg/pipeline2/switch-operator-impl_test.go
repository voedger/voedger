/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
 */

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwitchOperator_Close(t *testing.T) {
	closed := false
	operator := switchOperator[any]{
		branches: map[string]ISyncOperator[any]{"branch": mockSyncOp().
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
		operator := switchOperator[any]{
			branches: map[string]ISyncOperator[any]{
				"branch": NewSyncOp(func(ctx context.Context, work interface{}) (err error) {
					doSync = true
					return nil
				}),
			},
			switchLogic: mockSwitch[any]{func(work interface{}) (branch string, err error) {
				return "branch", nil
			}},
		}

		_ = operator.DoSync(context.Background(), nil)

		require.True(t, doSync)
	})
	t.Run("Should be not ok because switch logic return error", func(t *testing.T) {
		operator := switchOperator[any]{
			switchLogic: mockSwitch[any]{func(work interface{}) (branch string, err error) {
				return "branch", errors.New("switch logic error")
			}},
		}

		err := operator.DoSync(context.Background(), nil)

		require.Equal(t, "switch logic error", err.Error())
	})
}

func TestSwitchOperator(t *testing.T) {
	t.Run("Should create switch operator", func(t *testing.T) {
		switchOperator := SwitchOperator[any](mockSwitch[any]{}, SwitchBranch("branch", mockSyncOp().create()))

		require.NotNil(t, switchOperator)
	})

	t.Run("Should panic when switch logic is nil", func(t *testing.T) {
		require.PanicsWithValue(t, "switch must be not nil", func() {
			SwitchOperator[any](nil, SwitchBranch("branch", mockSyncOp().create()))
		})
	})
}

type mockSwitch[T any] struct {
	switchLogic func(work T) (branchName string, err error)
}

func (s mockSwitch[T]) Switch(work T) (branchName string, err error) {
	return s.switchLogic(work)
}
