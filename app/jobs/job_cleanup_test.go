// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

type fakeOldJobs struct {
	store.JobsStore
	err error
}

func (f fakeOldJobs) DeleteOld(int64) error {
	return f.err
}

type fakeSweeper struct{ swept int }

func (f *fakeSweeper) SweepPendingFormFiles(context.Context) error {
	f.swept++
	return nil
}

func TestCleanupJobSweepsPendingFormFilesEvenWhenOldJobsFail(t *testing.T) {
	sweeper := &fakeSweeper{}
	failure := errors.New("database gone")

	err := NewCleanupJob(fakeOldJobs{err: failure}, sweeper).Run(context.Background(), &model.Job{})

	if !errors.Is(err, failure) {
		t.Errorf("Run answered %v, want the old jobs' error", err)
	}
	if sweeper.swept != 1 {
		t.Errorf("swept %d times, want once", sweeper.swept)
	}
}

func TestCleanupJobWithoutASweeperDeletesOnlyOldJobs(t *testing.T) {
	failure := errors.New("database gone")

	err := NewCleanupJob(fakeOldJobs{err: failure}, nil).Run(context.Background(), &model.Job{})

	if !errors.Is(err, failure) {
		t.Errorf("Run answered %v, want the old jobs' error", err)
	}
}
