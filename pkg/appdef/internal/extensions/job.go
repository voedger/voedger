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

func NewJob(app appdef.IAppDef, ws *workspace, name QName) *Job {
	j := &Job{
		extension: makeExtension(app, ws, name, TypeKind_Job),
	}
	ws.appendType(j)
	return j
}

func (j Job) CronSchedule() string { return j.cronSchedule }

func (j *Job) setCronSchedule(cs string) { j.cronSchedule = cs }

// Validates job
//
// # Returns error:
//   - if cron schedule is invalid
func (j *Job) Validate() (err error) {
	err = j.extension.Validate()

	_, e := cron.ParseStandard(j.cronSchedule)
	if e != nil {
		err = errors.Join(err, enrichError(e, "%v cron schedule", j))
	}

	return err
}

// # Implements:
//   - IJobBuilder
type jobBuilder struct {
	extensionBuilder
	*Job
}

func newJobBuilder(j *Job) *jobBuilder {
	return &jobBuilder{
		extensionBuilder: makeExtensionBuilder(&j.extension),
		Job:              j,
	}
}

func (jb *jobBuilder) SetCronSchedule(cs string) IJobBuilder {
	jb.Job.setCronSchedule(cs)
	return jb
}
