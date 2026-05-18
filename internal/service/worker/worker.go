package worker

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	modeljob "github.com/ananthakumaran/paisa/internal/model/job"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
// move to. Terminal states (Completed, Failed) map to an empty slice.
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
// by the state machine. Must be called while holding the registry mutex.
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
	// Type identifies the recoverable task category (e.g. "sync").
	Type string `json:"-"`
	// Payload stores the serialized recoverable payload for restart recovery.
	Payload string `json:"-"`
	// Status is the current lifecycle state.
	Status JobStatus `json:"status"`
	// Error holds the error message when Status == StatusFailed; otherwise empty.
	Error string `json:"error,omitempty"`
	// Details holds per-step diagnostic messages accumulated during job execution.
	Details []string `json:"details,omitempty"`
	// ItemsCompleted is the number of items processed so far.
	ItemsCompleted int `json:"items_completed,omitempty"`
	// TotalItems is the total number of items to process.
	TotalItems int `json:"total_items,omitempty"`
	// CreatedAt is the wall-clock time at which the job was submitted.
	CreatedAt time.Time `json:"created_at"`
	// StartedAt is the wall-clock time at which the job began executing.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// FinishedAt is the wall-clock time at which the job reached a terminal state.
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	// Metadata holds arbitrary key-value pairs associated with the job.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DetailedJobFn is the function signature accepted by [Registry.SubmitDetailed].
//
// The progress argument is a thread-safe callback the function may call to
// report incremental progress. Each call sets ItemsCompleted and TotalItems
// on the Job so consumers can display a "X of Y" indicator.
type DetailedJobFn func(ctx context.Context, progress func(completed, total int)) (details []string, err error)

// RecoverableJobFn can resume a persisted job using the serialized payload.
type RecoverableJobFn func(ctx context.Context, payload json.RawMessage, progress func(completed, total int)) (details []string, err error)

// Registry is a thread-safe store for background Jobs. Use [NewRegistry] to
// create an instance; the zero value is not usable.
type Registry struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	subscribers map[int]chan Job
	nextSubID   int
	recoverable map[string]RecoverableJobFn
	db          *gorm.DB
}

// NewRegistry creates and returns an initialised Registry.
//
// When db is provided, job state is persisted in SQLite and any previously
// persisted jobs are loaded into memory for immediate API visibility.
func NewRegistry(db ...*gorm.DB) *Registry {
	var gormDB *gorm.DB
	if len(db) > 0 {
		gormDB = db[0]
	}

	r := &Registry{
		jobs:        make(map[string]*Job),
		subscribers: make(map[int]chan Job),
		recoverable: make(map[string]RecoverableJobFn),
		db:          gormDB,
	}
	if gormDB != nil {
		r.loadPersistedJobs()
	}
	return r
}

func (r *Registry) loadPersistedJobs() {
	var rows []modeljob.Job
	if err := r.db.Order("created_at asc").Find(&rows).Error; err != nil {
		log.WithError(err).Warn("worker: unable to load persisted jobs")
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range rows {
		job := fromModelJob(row)
		r.jobs[job.ID] = &job
	}
}

// RegisterRecoverable registers a recoverable job handler by type.
func (r *Registry) RegisterRecoverable(jobType string, fn RecoverableJobFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoverable[jobType] = fn
}

// RecoverInterrupted re-queues persisted pending/running jobs.
func (r *Registry) RecoverInterrupted(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	type recoverItem struct {
		id      string
		payload string
		fn      RecoverableJobFn
	}

	var toRecover []recoverItem

	r.mu.Lock()
	for _, job := range r.jobs {
		if job.Status != StatusPending && job.Status != StatusRunning {
			continue
		}
		fn := r.recoverable[job.Type]
		if fn == nil {
			now := time.Now().UTC()
			job.Status = StatusFailed
			job.Error = fmt.Sprintf("no recovery handler registered for job type %q", job.Type)
			job.FinishedAt = &now
			r.saveLocked(job)
			r.notifyLocked(cloneJob(*job))
			continue
		}

		if job.Status == StatusRunning {
			job.Status = StatusPending
			job.StartedAt = nil
			job.FinishedAt = nil
			job.Error = ""
		}
		r.saveLocked(job)
		r.notifyLocked(cloneJob(*job))
		toRecover = append(toRecover, recoverItem{id: job.ID, payload: job.Payload, fn: fn})
	}
	r.mu.Unlock()

	for _, item := range toRecover {
		payload := item.payload
		fn := item.fn
		go r.runDetailed(ctx, item.id, func(ctx context.Context, progress func(int, int)) ([]string, error) {
			return fn(ctx, json.RawMessage(payload), progress)
		})
	}

	return nil
}

// Submit enqueues fn as a new background job and returns the job ID.
func (r *Registry) Submit(ctx context.Context, fn func(ctx context.Context) error) string {
	id, err := r.submitDetailedInternal(ctx, nil, "", "", func(ctx context.Context, _ func(int, int)) ([]string, error) {
		return nil, fn(ctx)
	})
	if err != nil {
		log.WithError(err).Error("worker: failed to submit job")
		return ""
	}
	return id
}

// SubmitDetailed enqueues fn as a new background job and returns the job ID.
func (r *Registry) SubmitDetailed(ctx context.Context, metadata map[string]any, fn DetailedJobFn) string {
	id, err := r.submitDetailedInternal(ctx, metadata, "", "", fn)
	if err != nil {
		log.WithError(err).Error("worker: failed to submit detailed job")
		return ""
	}
	return id
}

// SubmitRecoverable submits a persisted job that can be replayed after restart.
func (r *Registry) SubmitRecoverable(
	ctx context.Context,
	jobType string,
	payload any,
	metadata map[string]any,
) (string, error) {
	r.mu.RLock()
	handler := r.recoverable[jobType]
	r.mu.RUnlock()
	if handler == nil {
		return "", fmt.Errorf("worker: no recoverable handler registered for type %q", jobType)
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("worker: marshal recoverable payload: %w", err)
	}

	return r.submitDetailedInternal(ctx, metadata, jobType, string(rawPayload), func(ctx context.Context, progress func(int, int)) ([]string, error) {
		return handler(ctx, rawPayload, progress)
	})
}

func (r *Registry) submitDetailedInternal(
	ctx context.Context,
	metadata map[string]any,
	jobType string,
	payload string,
	fn DetailedJobFn,
) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("worker: unable to generate job ID: %w", err)
	}

	job := &Job{
		ID:        id.String(),
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
		Metadata:  cloneMetadata(metadata),
	}

	r.mu.Lock()
	r.jobs[job.ID] = job
	if err := r.saveLocked(job); err != nil {
		delete(r.jobs, job.ID)
		r.mu.Unlock()
		return "", err
	}
	r.notifyLocked(cloneJob(*job))
	r.mu.Unlock()

	go r.runDetailed(ctx, job.ID, fn)

	return job.ID, nil
}

func (r *Registry) runDetailed(ctx context.Context, jobID string, fn DetailedJobFn) {
	now := time.Now().UTC()

	r.mu.Lock()
	job, ok := r.jobs[jobID]
	if !ok {
		r.mu.Unlock()
		return
	}
	transition(job, StatusRunning)
	job.StartedAt = &now
	job.FinishedAt = nil
	job.Error = ""
	r.saveLocked(job)
	r.notifyLocked(cloneJob(*job))
	r.mu.Unlock()

	log.WithField("job_id", jobID).Info("worker: job started")

	progress := func(completed, total int) {
		r.mu.Lock()
		defer r.mu.Unlock()
		job, ok := r.jobs[jobID]
		if !ok || job.Status.IsTerminal() {
			return
		}
		job.ItemsCompleted = completed
		job.TotalItems = total
		r.saveLocked(job)
		r.notifyLocked(cloneJob(*job))
	}

	details, runErr := fn(ctx, progress)
	finished := time.Now().UTC()

	r.mu.Lock()
	job, ok = r.jobs[jobID]
	if !ok {
		r.mu.Unlock()
		return
	}
	job.FinishedAt = &finished
	if len(details) > 0 {
		job.Details = append([]string(nil), details...)
	} else {
		job.Details = nil
	}
	if runErr != nil {
		transition(job, StatusFailed)
		job.Error = runErr.Error()
		log.WithField("job_id", job.ID).WithError(runErr).Warn("worker: job failed")
	} else {
		transition(job, StatusCompleted)
		job.Error = ""
		log.WithField("job_id", job.ID).Info("worker: job completed")
	}
	r.saveLocked(job)
	r.notifyLocked(cloneJob(*job))
	r.mu.Unlock()
}

// Get returns a snapshot of the Job with the given ID and true, or false when absent.
func (r *Registry) Get(id string) (Job, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[id]
	if !ok {
		return Job{}, false
	}
	clone := cloneJob(*j)
	return clone, true
}

// List returns all jobs ordered by CreatedAt ascending (oldest first).
func (r *Registry) List() []Job {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		result = append(result, cloneJob(*j))
	}
	slices.SortFunc(result, func(a, b Job) int {
		return cmp.Compare(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
	})
	return result
}

// Subscribe registers a listener for real-time job snapshots.
func (r *Registry) Subscribe() (<-chan Job, func()) {
	ch := make(chan Job, 32)

	r.mu.Lock()
	subID := r.nextSubID
	r.nextSubID++
	r.subscribers[subID] = ch
	r.mu.Unlock()

	cancel := func() {
		r.mu.Lock()
		if c, ok := r.subscribers[subID]; ok {
			delete(r.subscribers, subID)
			close(c)
		}
		r.mu.Unlock()
	}

	return ch, cancel
}

func (r *Registry) saveLocked(job *Job) error {
	if r.db == nil {
		return nil
	}
	if err := r.db.Save(toModelJob(*job)).Error; err != nil {
		log.WithError(err).WithField("job_id", job.ID).Warn("worker: failed to persist job")
		return err
	}
	return nil
}

func (r *Registry) notifyLocked(job Job) {
	for _, sub := range r.subscribers {
		select {
		case sub <- job:
		default:
		}
	}
}

func toModelJob(job Job) *modeljob.Job {
	return &modeljob.Job{
		ID:             job.ID,
		Type:           job.Type,
		Status:         string(job.Status),
		Error:          job.Error,
		Details:        append([]string(nil), job.Details...),
		ItemsCompleted: job.ItemsCompleted,
		TotalItems:     job.TotalItems,
		Metadata:       cloneMetadata(job.Metadata),
		Payload:        job.Payload,
		CreatedAt:      job.CreatedAt,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
	}
}

func fromModelJob(row modeljob.Job) Job {
	status := JobStatus(row.Status)
	if status == "" {
		status = StatusPending
	}
	return Job{
		ID:             row.ID,
		Type:           row.Type,
		Payload:        row.Payload,
		Status:         status,
		Error:          row.Error,
		Details:        append([]string(nil), row.Details...),
		ItemsCompleted: row.ItemsCompleted,
		TotalItems:     row.TotalItems,
		Metadata:       cloneMetadata(row.Metadata),
		CreatedAt:      row.CreatedAt,
		StartedAt:      row.StartedAt,
		FinishedAt:     row.FinishedAt,
	}
}

func cloneMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneJob(in Job) Job {
	out := in
	if len(in.Details) > 0 {
		out.Details = append([]string(nil), in.Details...)
	} else {
		out.Details = nil
	}
	out.Metadata = cloneMetadata(in.Metadata)
	return out
}

var ErrNoRecoverableHandler = errors.New("worker: no recoverable handler")
