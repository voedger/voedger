/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
 */

package pipeline

import "context"

type switchOperator[T any] struct {
	switchLogic       ISwitch[T]
	branches          map[string]ISyncOperator[T]
	currentBranchName string
}

func (s switchOperator[T]) Close() {
	for _, branch := range s.branches {
		branch.Close()
	}
}

func (s switchOperator[T]) DoSync(ctx context.Context, work T) (err error) {
	s.currentBranchName, err = s.switchLogic.Switch(work)
	if err != nil {
		return err
	}
	return s.branches[s.currentBranchName].DoSync(ctx, work)
}

type SwitchOperatorOptionFunc[T any] func(*switchOperator[T])

func SwitchOperator[T any](switchLogic ISwitch[T], branch SwitchOperatorOptionFunc[T], branches ...SwitchOperatorOptionFunc[T]) ISyncOperator[T] {
	if switchLogic == nil {
		panic("switch must be not nil")
	}
	switchOperator := &switchOperator[T]{
		switchLogic: switchLogic,
		branches:    make(map[string]ISyncOperator[T]),
	}
	branch(switchOperator)
	for _, branch := range branches {
		branch(switchOperator)
	}
	return switchOperator
}

func SwitchBranch[T any](name string, operator ISyncOperator[T]) SwitchOperatorOptionFunc[T] {
	return func(switchOperator *switchOperator[T]) {
		switchOperator.branches[name] = operator
	}
}
