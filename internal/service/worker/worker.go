package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

// JobStatus represents the lifecycle state of a background Job.
type JobStatus string

const (
	// StatusPending means the job has been submitted but has not yet started.
	StatusPending JobStatus = "pending"
	// StatusRunning means the job is actively executing.
	StatusRunning JobStatus = "running"
	// StatusCompleted means the job finished successfully.
	StatusCompleted JobStatus = "completed"
	// StatusFailed means the job finished with an error.
	StatusFailed JobStatus = "failed"
)

// validTransitions maps every JobStatus to the set of states it may legally
// move to.  Terminal states (Completed, Failed) map to an empty slice.
var validTransitions = map[JobStatus][]JobStatus{
	StatusPending:   {StatusRunning},
	StatusRunning:   {StatusCompleted, StatusFailed},
	StatusCompleted: {},
	StatusFailed:    {},
}

// IsTerminal reports whether s is a terminal state (Completed or Failed).
// A Job in a terminal state will never change state again.
func (s JobStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed
}

// transition moves job to next, panicking if the transition is not permitted
// by the state machine.  Must be called while holding the registry mutex.
func transition(job *Job, next JobStatus) {
	for _, allowed := range validTransitions[job.Status] {
		if allowed == next {
			job.Status = next
			return
		}
	}
	panic(fmt.Sprintf("worker: invalid state transition %s → %s for job %s", job.Status, next, job.ID))
}

// Job holds all observable state for a single unit of background work.
type Job struct {
	// ID is the unique, opaque identifier assigned at submission time.
	ID string `json:"id"`
	// Status is the current lifecycle state.
	Status JobStatus `json:"status"`
	// Error holds the error message when Status == StatusFailed; otherwise empty.
	Error string `json:"error,omitempty"`
	// CreatedAt is the wall-clock time at which the job was submitted.
	CreatedAt time.Time `json:"created_at"`
	// StartedAt is the wall-clock time at which the job began executing; zero if still pending.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// FinishedAt is the wall-clock time at which the job reached a terminal state; zero if not yet finished.
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// Registry is a thread-safe store for background Jobs.  Use [NewRegistry] to
// create an instance; the zero value is not usable.
type Registry struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewRegistry creates and returns an initialised, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		jobs: make(map[string]*Job),
	}
}

// Submit enqueues fn as a new background job and returns the job ID.
// The function receives a context that is cancelled when the process exits; it
// should respect cancellation for clean shutdown.
//
// Submit is non-blocking: fn is executed asynchronously in a separate goroutine.
func (r *Registry) Submit(ctx context.Context, fn func(ctx context.Context) error) string {
	id, err := uuid.NewV4()
	if err != nil {
		// uuid.NewV4 reads from crypto/rand; failure is only possible if the OS
		// random source is exhausted, which is unrecoverable.
		log.WithError(err).Fatal("worker: unable to generate job ID")
	}

	job := &Job{
		ID:        id.String(),
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}

	r.mu.Lock()
	r.jobs[job.ID] = job
	r.mu.Unlock()

	go r.run(ctx, job, fn)

	return job.ID
}

// run executes fn inside a goroutine and updates job state accordingly.
func (r *Registry) run(ctx context.Context, job *Job, fn func(ctx context.Context) error) {
	now := time.Now().UTC()

	r.mu.Lock()
	transition(job, StatusRunning)
	job.StartedAt = &now
	r.mu.Unlock()

	log.WithField("job_id", job.ID).Info("worker: job started")

	runErr := fn(ctx)

	finished := time.Now().UTC()

	r.mu.Lock()
	job.FinishedAt = &finished
	if runErr != nil {
		transition(job, StatusFailed)
		job.Error = runErr.Error()
		log.WithField("job_id", job.ID).WithError(runErr).Warn("worker: job failed")
	} else {
		transition(job, StatusCompleted)
		log.WithField("job_id", job.ID).Info("worker: job completed")
	}
	r.mu.Unlock()
}

// Get returns a snapshot of the Job with the given ID and true, or the zero
// Job value and false when no such job exists.
func (r *Registry) Get(id string) (Job, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[id]
	if !ok {
		return Job{}, false
	}
	return *j, true
}

// List returns a slice containing a snapshot of every Job known to the
// registry.  The order is not guaranteed.
func (r *Registry) List() []Job {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		result = append(result, *j)
	}
	return result
}
