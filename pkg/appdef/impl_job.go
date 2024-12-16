/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"

	"github.com/robfig/cron/v3"
)

// # Implements:
//   - IJob
type job struct {
	extension
	cronSchedule string
}

func newJob(app *appDef, ws *workspace, name QName) *job {
	j := &job{
		extension: makeExtension(app, ws, name, TypeKind_Job),
	}
	ws.appendType(j)
	return j
}

func (j job) CronSchedule() string { return j.cronSchedule }

func (j *job) setCronSchedule(cs string) { j.cronSchedule = cs }

// Validates job
//
// # Returns error:
//   - if cron schedule is invalid
func (j *job) Validate() (err error) {
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
	*job
}

func newJobBuilder(j *job) *jobBuilder {
	return &jobBuilder{
		extensionBuilder: makeExtensionBuilder(&j.extension),
		job:              j,
	}
}

func (jb *jobBuilder) SetCronSchedule(cs string) IJobBuilder {
	jb.job.setCronSchedule(cs)
	return jb
}
