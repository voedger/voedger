/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package states

const (
	UndefinedAttribute AttributeKind = iota - 1
	DockerStackAttribute
	EdgerAttribute
	CommandAttribute

	AttributeKindCount
)

var (
	AttributeKindNames [AttributeKindCount]string = [AttributeKindCount]string{
		"DockerStackAttribute",
		"EdgerAttribute",
		"CommandAttribute",
	}
)

const (
	UndefinedStatus ActualStatus = iota - 1
	PendingStatus
	InProgressStatus
	FinishedStatus

	ActualStatusCount
)

var (
	ActualStatusNames [ActualStatusCount]string = [ActualStatusCount]string{
		"PendingStatus",
		"InProgressStatus",
		"FinishedStatus",
	}
)
