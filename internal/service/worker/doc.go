// Package worker provides background job execution infrastructure for the
// Paisa application.
//
// # Overview
//
// The worker package centres around two types:
//
//   - [Job] – a unit of background work that progresses through a defined state
//     machine (Pending → Running → Completed | Failed).
//   - [Registry] – a thread-safe store that tracks every Job submitted to the
//     process.  Callers submit a function via [Registry.Submit], receive a
//     unique job ID back, and can later query status via [Registry.Get] or
//     retrieve the full job list via [Registry.List].
//
// # Lifecycle
//
// A newly created Job starts in the [StatusPending] state.  When the registry
// dispatches it, the Job transitions to [StatusRunning].  The worker goroutine
// then calls the provided function and, depending on whether an error is
// returned, moves the Job to either [StatusCompleted] or [StatusFailed].
//
// State transitions are monotonic: once a Job reaches a terminal state
// (Completed or Failed) its status never changes again.  Use
// [JobStatus.IsTerminal] to test whether a given status is terminal.
//
// All legal transitions are enumerated in the package-level validTransitions
// map.  The transition helper enforces this map at runtime: any attempt to
// move a Job to an unlisted next state causes a panic, making invalid
// transitions immediately observable during development and testing.
//
// # Concurrency
//
// All exported methods on [Registry] are safe for concurrent use from multiple
// goroutines.  The internal job map is protected by a read/write mutex so that
// read-heavy polling (e.g. from the API layer) does not serialise with new
// submissions.
//
// # Boundaries
//
// The worker package is responsible only for lifecycle management and status
// tracking.  Business logic (syncing the journal, scraping prices, etc.) lives
// in the packages that own those concerns and is passed to [Registry.Submit] as
// a plain Go function.
package worker
