// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

const (
	jobRetention    = 24 * time.Hour
	cleanupInterval = time.Hour
)

type PendingFileSweeper interface {
	SweepPendingFormFiles(ctx context.Context) error
}

type CleanupJob struct {
	store   store.JobsStore
	sweeper PendingFileSweeper
}

// NewCleanupJob gives a job that deletes old jobs and, when sweeper is not
// nil, old pending form uploads.
func NewCleanupJob(store store.JobsStore, sweeper PendingFileSweeper) CleanupJob {
	return CleanupJob{store: store, sweeper: sweeper}
}

func (j CleanupJob) Type() string {
	return model.JobTypeCleanupJobs
}

func (j CleanupJob) Schedule() (time.Duration, bool) {
	return cleanupInterval, true
}

func (j CleanupJob) Run(ctx context.Context, job *model.Job) error {
	jobsErr := j.store.DeleteOld(time.Now().Add(-jobRetention).Unix())
	if j.sweeper == nil {
		return jobsErr
	}

	return errors.Join(jobsErr, j.sweeper.SweepPendingFormFiles(ctx))
}
