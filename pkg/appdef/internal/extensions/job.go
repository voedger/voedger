/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"errors"

	"github.com/robfig/cron/v3"
	"github.com/voedger/voedger/pkg/appdef"
)

// # Supports:
//   - appdef.IJob
type Job struct {
	Extension
	cronSchedule string
}

func NewJob(ws appdef.IWorkspace, name appdef.QName) *Job {
	return &Job{Extension: MakeExtension(ws, name, appdef.TypeKind_Job)}
}

func (j Job) CronSchedule() string { return j.cronSchedule }

func (j *Job) setCronSchedule(cs string) { j.cronSchedule = cs }

// Validates job
//
// # Returns error:
//   - if cron schedule is invalid
func (j *Job) Validate() (err error) {
	err = j.Extension.Validate()

	_, e := cron.ParseStandard(j.cronSchedule)
	if e != nil {
		err = errors.Join(err, appdef.EnrichError(e, "%v cron schedule", j))
	}

	return err
}

// # Supports:
//   - appdef.IJobBuilder
type JobBuilder struct {
	ExtensionBuilder
	*Job
}

func NewJobBuilder(j *Job) *JobBuilder {
	return &JobBuilder{
		ExtensionBuilder: MakeExtensionBuilder(&j.Extension),
		Job:              j,
	}
}

func (jb *JobBuilder) SetCronSchedule(cs string) appdef.IJobBuilder {
	jb.Job.setCronSchedule(cs)
	return jb
}
