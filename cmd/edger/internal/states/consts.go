/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
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
