/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Job is a extension that executes by schedule.
type IJob interface {
	IExtension

	// Schedule to trigger projector as cron expression.
	CronSchedule() string
}

type IJobBuilder interface {
	IExtensionBuilder

	// Set schedule to trigger projector as cron expression.
	SetCronSchedule(string) IJobBuilder
}

type IJobsBuilder interface {
	// Add new job
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddJob(QName) IJobBuilder
}
