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
	operator := forkOperator[any]{
		branches: []ISyncOperator[any]{mockSyncOp().
			close(func() {
				closeDone = true
			}).create()},
	}

	operator.Close()

	require.True(t, closeDone)
}

func TestForkOperator_DoSync(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		operator := forkOperator[testwork]{
			fork: ForkSame[testwork],
			branches: []ISyncOperator[testwork]{
				NewSyncOp[testwork](func(ctx context.Context, work testwork) (err error) {
					return nil
				}),
			},
		}

		err := operator.DoSync(context.Background(), newTestWork())

		require.NoError(t, err)
	})
	t.Run("Should be not ok because fork logic return error", func(t *testing.T) {
		testErr := errors.New("fork error")
		operator := forkOperator[testwork]{
			fork: func(work testwork, branchNumber int) (fork testwork, err error) {
				return testwork{}, testErr
			},
			branches: []ISyncOperator[testwork]{nil},
		}

		err := operator.DoSync(context.Background(), newTestWork())

		require.ErrorIs(t, err, testErr)
	})
	t.Run("Should panic when fork returns nil", func(t *testing.T) {
		operator := forkOperator[testwork]{
			fork: func(work testwork, branchNumber int) (fork testwork, err error) {
				return testwork{}, nil
			},
			branches: []ISyncOperator[testwork]{nil},
		}

		require.PanicsWithValue(t, "fork is nil", func() {
			_ = operator.DoSync(context.Background(), newTestWork())
		})
	})
	t.Run("Should be not ok because branch DoSync() return error", func(t *testing.T) {
		operator := forkOperator[testwork]{
			fork: ForkSame[testwork],
			branches: []ISyncOperator[testwork]{
				NewSyncOp(func(ctx context.Context, work testwork) (err error) {
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
		forkOperator := ForkOperator(ForkSame[any], ForkBranch(mockSyncOp().create()))

		require.NotNil(t, forkOperator)
	})

	t.Run("Should panic when fork logic is nil", func(t *testing.T) {
		require.PanicsWithValue(t, "fork must be not nil", func() {
			ForkOperator[any](nil, ForkBranch(mockSyncOp().create()))
		})
	})
}
