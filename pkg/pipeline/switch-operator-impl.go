/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
 */

package pipeline

import "context"

type switchOperator struct {
	switchLogic       ISwitch
	branches          map[string]ISyncOperator
	currentBranchName string
}

func (s switchOperator) Close() {
	for _, branch := range s.branches {
		branch.Close()
	}
}

func (s switchOperator) DoSync(ctx context.Context, work interface{}) (err error) {
	s.currentBranchName, err = s.switchLogic.Switch(work)
	if err != nil {
		return err
	}
	return s.branches[s.currentBranchName].DoSync(ctx, work)
}

type SwitchOperatorOptionFunc func(*switchOperator)

func SwitchOperator(switchLogic ISwitch, branch SwitchOperatorOptionFunc, branches ...SwitchOperatorOptionFunc) ISyncOperator {
	if switchLogic == nil {
		panic("switch must be not nil")
	}
	switchOperator := &switchOperator{
		switchLogic: switchLogic,
		branches:    make(map[string]ISyncOperator),
	}
	branch(switchOperator)
	for _, branch := range branches {
		branch(switchOperator)
	}
	return switchOperator
}

func SwitchBranch(name string, operator ISyncOperator) SwitchOperatorOptionFunc {
	return func(switchOperator *switchOperator) {
		switchOperator.branches[name] = operator
	}
}
