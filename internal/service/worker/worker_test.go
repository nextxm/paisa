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

// TestList_OrderedByCreatedAt verifies that List returns jobs sorted by
// CreatedAt in ascending order (oldest first), regardless of the order in
// which goroutines complete.
func TestList_OrderedByCreatedAt(t *testing.T) {
	r := worker.NewRegistry()
	const n = 5

	// Gate the workers so none finish before we call List, ensuring we can
	// rely on CreatedAt ordering rather than completion ordering.
	gate := make(chan struct{})
	for range n {
		r.Submit(context.Background(), func(_ context.Context) error {
			<-gate
			return nil
		})
		// 1 ms gap ensures strictly increasing CreatedAt values on any OS.
		time.Sleep(time.Millisecond)
	}

	jobs := r.List()
	require.Len(t, jobs, n)

	for i := 1; i < len(jobs); i++ {
		assert.True(t,
			!jobs[i].CreatedAt.Before(jobs[i-1].CreatedAt),
			"job[%d].CreatedAt (%v) must be >= job[%d].CreatedAt (%v)",
			i, jobs[i].CreatedAt, i-1, jobs[i-1].CreatedAt,
		)
	}

	close(gate)
}

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

// TestRegistry_ConcurrentList verifies that calling List concurrently with
// Submit causes no data races.  Run with -race.
func TestRegistry_ConcurrentList(t *testing.T) {
	r := worker.NewRegistry()
	const goroutines = 30

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			r.Submit(context.Background(), func(_ context.Context) error {
				return nil
			})
			_ = r.List()
		}()
	}

	wg.Wait()
}

// TestRegistry_ConcurrentSubmitGetList is a high-fan-out stress test that
// hammers Submit, Get, and List from many goroutines simultaneously to
// surface any data race.  Run with -race.
func TestRegistry_ConcurrentSubmitGetList(t *testing.T) {
	r := worker.NewRegistry()
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			id := r.Submit(context.Background(), func(_ context.Context) error {
				return nil
			})
			r.Get(id)
			r.List()
		}()
	}

	wg.Wait()
}

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

// ---------------------------------------------------------------------------
// SubmitDetailed tests
// ---------------------------------------------------------------------------

// TestSubmitDetailed_ReturnsNonEmptyID verifies that SubmitDetailed always
// returns a non-empty job ID.
func TestSubmitDetailed_ReturnsNonEmptyID(t *testing.T) {
	r := worker.NewRegistry()
	id := r.SubmitDetailed(context.Background(), nil, func(_ context.Context) ([]string, error) {
		return nil, nil
	})
	assert.NotEmpty(t, id, "SubmitDetailed must return a non-empty job ID")
}

// TestSubmitDetailed_Completed_NoDetails verifies that when the function
// returns no details and no error the job reaches Completed with an empty
// Details slice.
func TestSubmitDetailed_Completed_NoDetails(t *testing.T) {
	r := worker.NewRegistry()
	id := r.SubmitDetailed(context.Background(), nil, func(_ context.Context) ([]string, error) {
		return nil, nil
	})

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusCompleted
	}, 2*time.Second, 5*time.Millisecond, "job must reach Completed")

	job, ok := r.Get(id)
	require.True(t, ok)
	assert.Equal(t, worker.StatusCompleted, job.Status)
	assert.Empty(t, job.Details, "Details must be empty when no details were returned")
	assert.Empty(t, job.Error)
}

// TestSubmitDetailed_Completed_WithDetails verifies that details returned by
// the function are stored in Job.Details and the job still reaches Completed
// (details do not cause the job to fail).
func TestSubmitDetailed_Completed_WithDetails(t *testing.T) {
	r := worker.NewRegistry()
	want := []string{"commodity A failed", "commodity B failed"}

	id := r.SubmitDetailed(context.Background(), nil, func(_ context.Context) ([]string, error) {
		return want, nil
	})

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusCompleted
	}, 2*time.Second, 5*time.Millisecond, "job must reach Completed")

	job, ok := r.Get(id)
	require.True(t, ok)
	assert.Equal(t, worker.StatusCompleted, job.Status)
	assert.Equal(t, want, job.Details, "Details must match the messages returned by the function")
}

// TestSubmitDetailed_Failed_WithDetails verifies that when the function returns
// both details and an error the job reaches Failed, the Error field is populated,
// and Details are also stored so operators see both the top-level failure and the
// per-step messages.
func TestSubmitDetailed_Failed_WithDetails(t *testing.T) {
	r := worker.NewRegistry()
	sentinelErr := errors.New("sync failed")
	want := []string{"XIRR did not converge for account: Assets:Equity:AAPL"}

	id := r.SubmitDetailed(context.Background(), nil, func(_ context.Context) ([]string, error) {
		return want, sentinelErr
	})

	assert.Eventually(t, func() bool {
		j, _ := r.Get(id)
		return j.Status == worker.StatusFailed
	}, 2*time.Second, 5*time.Millisecond, "job must reach Failed")

	job, ok := r.Get(id)
	require.True(t, ok)
	assert.Equal(t, worker.StatusFailed, job.Status)
	assert.Equal(t, sentinelErr.Error(), job.Error)
	assert.Equal(t, want, job.Details, "Details must be stored even when the job fails")
}
