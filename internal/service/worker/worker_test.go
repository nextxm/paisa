package worker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/service/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Basic lifecycle tests
// ---------------------------------------------------------------------------

// TestSubmit_ReturnsNonEmptyID verifies that Submit always returns a non-empty
// job ID.
func TestSubmit_ReturnsNonEmptyID(t *testing.T) {
	r := worker.NewRegistry()
	id := r.Submit(context.Background(), func(_ context.Context) error {
		return nil
	})
	assert.NotEmpty(t, id, "Submit must return a non-empty job ID")
}

// TestSubmit_JobInitiallyPendingOrRunning verifies that immediately after
// Submit the job exists and is either Pending or Running (it may transition
// almost instantly on fast machines).
func TestSubmit_JobInitiallyPendingOrRunning(t *testing.T) {
	r := worker.NewRegistry()

	// Use a channel to gate the worker function so we can observe Pending/Running.
	gate := make(chan struct{})
	id := r.Submit(context.Background(), func(_ context.Context) error {
		<-gate
		return nil
	})

	job, ok := r.Get(id)
	require.True(t, ok, "job must be retrievable immediately after Submit")
	assert.True(t,
		job.Status == worker.StatusPending || job.Status == worker.StatusRunning,
		"job must be Pending or Running immediately after Submit, got %s", job.Status)

	close(gate) // allow the worker to finish
	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusCompleted
	}, 2*time.Second, 5*time.Millisecond, "job must reach Completed after gate is opened")
}

// TestJobLifecycle_Completed verifies the full happy-path state transition:
// Pending → Running → Completed.
func TestJobLifecycle_Completed(t *testing.T) {
	r := worker.NewRegistry()

	started := make(chan struct{})
	id := r.Submit(context.Background(), func(_ context.Context) error {
		close(started)
		return nil
	})

	// Wait until the function starts (confirms Running state was set).
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job function never started")
	}

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusCompleted
	}, 2*time.Second, 5*time.Millisecond, "job must reach Completed")

	job, ok := r.Get(id)
	require.True(t, ok)
	assert.Equal(t, worker.StatusCompleted, job.Status)
	assert.Empty(t, job.Error)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.FinishedAt)
	assert.True(t, !job.CreatedAt.IsZero())
}

// TestJobLifecycle_Failed verifies the error-path state transition:
// Pending → Running → Failed.
func TestJobLifecycle_Failed(t *testing.T) {
	r := worker.NewRegistry()
	sentinelErr := errors.New("something went wrong")

	id := r.Submit(context.Background(), func(_ context.Context) error {
		return sentinelErr
	})

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusFailed
	}, 2*time.Second, 5*time.Millisecond, "job must reach Failed")

	job, ok := r.Get(id)
	require.True(t, ok)
	assert.Equal(t, worker.StatusFailed, job.Status)
	assert.Equal(t, sentinelErr.Error(), job.Error)
	assert.NotNil(t, job.FinishedAt)
}

// TestGet_UnknownID verifies that Get returns (zero, false) for an unknown ID.
func TestGet_UnknownID(t *testing.T) {
	r := worker.NewRegistry()
	_, ok := r.Get("does-not-exist")
	assert.False(t, ok, "Get on unknown ID must return false")
}

// TestList_Empty verifies that List returns an empty (non-nil) slice when no
// jobs have been submitted.
func TestList_Empty(t *testing.T) {
	r := worker.NewRegistry()
	jobs := r.List()
	assert.NotNil(t, jobs)
	assert.Empty(t, jobs)
}

// TestList_ContainsAllSubmittedJobs verifies that List returns all submitted
// jobs.
func TestList_ContainsAllSubmittedJobs(t *testing.T) {
	r := worker.NewRegistry()
	const n = 5
	gate := make(chan struct{})
	ids := make([]string, n)
	for i := range n {
		ids[i] = r.Submit(context.Background(), func(_ context.Context) error {
			<-gate
			return nil
		})
	}

	jobs := r.List()
	assert.Len(t, jobs, n, "List must include all submitted jobs")

	close(gate)
}

// TestSubmit_IDs_AreUnique verifies that every submitted job gets a distinct ID.
func TestSubmit_IDs_AreUnique(t *testing.T) {
	r := worker.NewRegistry()
	const n = 20
	seen := make(map[string]struct{}, n)
	for range n {
		id := r.Submit(context.Background(), func(_ context.Context) error { return nil })
		_, dup := seen[id]
		assert.False(t, dup, "duplicate job ID detected: %s", id)
		seen[id] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

// TestSubmit_ContextCancellation verifies that the context passed to the job
// function is cancelled when the parent context is cancelled, and the job is
// eventually marked Failed.
func TestSubmit_ContextCancellation(t *testing.T) {
	r := worker.NewRegistry()

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	id := r.Submit(ctx, func(jobCtx context.Context) error {
		close(started)
		<-jobCtx.Done()
		return jobCtx.Err()
	})

	<-started // ensure the job has entered the running state
	cancel()  // trigger cancellation

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusFailed
	}, 2*time.Second, 5*time.Millisecond, "job must reach Failed after context cancellation")

	job, _ := r.Get(id)
	assert.Equal(t, worker.StatusFailed, job.Status)
	assert.NotEmpty(t, job.Error)
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// State machine helpers
// ---------------------------------------------------------------------------

// TestIsTerminal verifies that only Completed and Failed are terminal states.
func TestIsTerminal(t *testing.T) {
	assert.False(t, worker.StatusPending.IsTerminal(), "Pending must not be terminal")
	assert.False(t, worker.StatusRunning.IsTerminal(), "Running must not be terminal")
	assert.True(t, worker.StatusCompleted.IsTerminal(), "Completed must be terminal")
	assert.True(t, worker.StatusFailed.IsTerminal(), "Failed must be terminal")
	// An unknown/invalid status value must not be treated as terminal.
	assert.False(t, worker.JobStatus("unknown").IsTerminal(), "unknown status must not be terminal")
}

// TestJobReachesTerminalState verifies that once a job reaches a terminal
// state it is not modified again, confirming state monotonicity.
func TestJobReachesTerminalState_NeverChanges(t *testing.T) {
	r := worker.NewRegistry()

	id := r.Submit(context.Background(), func(_ context.Context) error {
		return nil
	})

	// Wait for terminal state.
	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status.IsTerminal()
	}, 2*time.Second, 5*time.Millisecond, "job must reach a terminal state")

	first, _ := r.Get(id)
	// A short sleep gives any stray goroutine a chance to modify the job (it
	// should not).
	time.Sleep(20 * time.Millisecond)
	second, _ := r.Get(id)

	assert.Equal(t, first.Status, second.Status, "terminal status must not change")
	assert.Equal(t, first.FinishedAt, second.FinishedAt, "FinishedAt must not change after terminal state")
}

// TestRegistry_ConcurrentSubmitAndGet verifies that multiple goroutines can
// call Submit and Get simultaneously without data races.  Run with -race.
func TestRegistry_ConcurrentSubmitAndGet(t *testing.T) {
	r := worker.NewRegistry()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			id := r.Submit(context.Background(), func(_ context.Context) error {
				return nil
			})
			// Also exercise Get and List concurrently.
			r.Get(id)
			r.List()
		}()
	}

	wg.Wait()
}
