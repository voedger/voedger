/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package pipeline

import "context"

type asyncSwitchOperator struct {
	switchLogic ISwitch
	branches    map[string]IAsyncPipeline
	AsyncNOOP
}

func AsyncSwitchOperator(switchLogic ISwitch, firstBranch AsyncSwitchOperatorOptionFunc, otherBranches ...AsyncSwitchOperatorOptionFunc) IAsyncOperator {
	res := &asyncSwitchOperator{
		switchLogic: switchLogic,
		branches:    make(map[string]IAsyncPipeline)}
	firstBranch(res)
	for _, branch := range otherBranches {
		branch(res)
	}
	return res
}

type AsyncSwitchOperatorOptionFunc func(*asyncSwitchOperator)

func (as *asyncSwitchOperator) DoAsync(_ context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
	branchName, err := as.switchLogic.Switch(work)
	if err != nil {
		return work, err
	}
	return nil, as.branches[branchName].SendAsync(work)
}

func AsyncSwitchBranch(name string, branch IAsyncPipeline) AsyncSwitchOperatorOptionFunc {
	return func(as *asyncSwitchOperator) {
		as.branches[name] = branch
	}
}
