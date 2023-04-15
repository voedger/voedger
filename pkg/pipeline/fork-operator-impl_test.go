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

func TestForkOperator_Close(t *testing.T) {
	closeDone := false
	operator := forkOperator{
		branches: []ISyncOperator{mockSyncOp().
			close(func() {
				closeDone = true
			}).create()},
	}

	operator.Close()

	require.True(t, closeDone)
}

func TestForkOperator_DoSync(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		operator := forkOperator{
			fork: ForkSame,
			branches: []ISyncOperator{
				NewSyncOp(func(ctx context.Context, work interface{}) (err error) {
					return nil
				}),
			},
		}

		err := operator.DoSync(context.Background(), newTestWork())

		require.Nil(t, err)
	})
	t.Run("Should be not ok because fork logic return error", func(t *testing.T) {
		testErr := errors.New("fork error")
		operator := forkOperator{
			fork: func(work interface{}, branchNumber int) (fork interface{}, err error) {
				return nil, testErr
			},
			branches: []ISyncOperator{nil},
		}

		err := operator.DoSync(context.Background(), newTestWork())

		require.ErrorIs(t, err, testErr)
	})
	t.Run("Should panic when fork returns nil", func(t *testing.T) {
		operator := forkOperator{
			fork: func(work interface{}, branchNumber int) (fork interface{}, err error) {
				return nil, nil
			},
			branches: []ISyncOperator{nil},
		}

		require.PanicsWithValue(t, "fork is nil", func() {
			_ = operator.DoSync(context.Background(), newTestWork())
		})
	})
	t.Run("Should be not ok because branch DoSync() return error", func(t *testing.T) {
		operator := forkOperator{
			fork: ForkSame,
			branches: []ISyncOperator{
				NewSyncOp(func(ctx context.Context, work interface{}) (err error) {
					return errors.New("branch DoSync() error")
				}),
			},
		}

		err := operator.DoSync(context.Background(), newTestWork())

		require.IsType(t, ErrInBranches{}, err)
		require.Equal(t, "branch DoSync() error", err.Error())
	})
}

func TestForkOperator(t *testing.T) {
	t.Run("Should create fork operator", func(t *testing.T) {
		forkOperator := ForkOperator(ForkSame, ForkBranch(mockSyncOp().create()))

		require.NotNil(t, forkOperator)
	})

	t.Run("Should panic when fork logic is nil", func(t *testing.T) {
		require.PanicsWithValue(t, "fork must be not nil", func() {
			ForkOperator(nil, ForkBranch(mockSyncOp().create()))
		})
	})
}
