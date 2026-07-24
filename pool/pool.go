// Package pool provides a generic worker pool capable of executing heterogeneous jobs.
// Jobs may have arbitrary argument and result types. The pool itself is untyped and
// operates on the Job interface, allowing different kinds of work to coexist in the
// same pool.
//
// Channel ownership rules:
//   - The caller owns any result channels passed into jobs.
//   - Jobs MUST NOT close shared result channels.
//   - The caller is responsible for synchronization and channel closure.

package pool

import (
	"context"
	"fmt"
	"sync"
)

// Func represents a unit of work that accepts arguments and returns an error.
// It does not produce a result.
type Func[PoolArgs any] func(id string, args PoolArgs) error

// FuncWithResult represents a unit of work that accepts arguments and returns
// a result along with an error.
type FuncWithResult[PoolArgs, PoolResult any] func(id string, args PoolArgs) (PoolResult, error)

// Job represents a unit of work executed by a worker.
// Implementations must be safe to call from a worker goroutine.
type Job interface {
	Run()
}

// jobWithResult is a Job that executes a function and sends its result to a caller-owned
// result channel.
//
// IMPORTANT:
//   - resultChan is owned by the caller.
//   - jobWithResult MUST NOT close resultChan, as multiple jobs may share it.
//   - Errors are reported by logging; no value is sent on error.
type jobWithResult[PoolArgs, PoolResult any] struct {
	id         string
	resultChan chan PoolResult
	args       PoolArgs
	fn         FuncWithResult[PoolArgs, PoolResult]
}

// Run executes the job function and sends the result to the result channel.
// If the function returns an error, it is logged and no result is sent.
func (j *jobWithResult[PoolArgs, PoolResult]) Run() {
	result, err := j.fn(j.id, j.args)
	if err != nil {
		fmt.Println("error running job", err)
		return
	}
	j.resultChan <- result
}

// job is a Job that executes a function without producing a result.
type job[PoolArgs any] struct {
	id   string
	args PoolArgs
	fn   Func[PoolArgs]
}

// Run executes the job function and logs any returned error.
func (j *job[PoolArgs]) Run() {
	if err := j.fn(j.id, j.args); err != nil {
		fmt.Println("error running job", err)
	}
}

// worker executes jobs received from the shared job channel.
// A worker runs until the job channel is closed.
type worker struct {
	id         int
	jobChannel chan Job
}

// Start begins the worker loop. It blocks until the job channel is closed.
func (w *worker) Start() {
	for job := range w.jobChannel {
		job.Run()
	}
}

// AddJob submits a job to the pool.
// If the context is canceled before the job can be enqueued, the job is discarded.
// This call may block if the job queue is full.
func (p *Pool) AddJob(ctx context.Context, job Job) {
	select {
	case <-p.ctx.Done():
		return
	case <-ctx.Done():
		return
	case p.jobChannel <- job:
	}

}

// Close closes the job channel, signaling all workers to exit once
// all queued jobs have been processed.
// It also waits for all workers to finish
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		// Canceling the context will cause AddJob to return
		p.cancel()
		// Closing the job channel will cause workers to exit
		close(p.jobChannel)
	})
	// Wait for all workers to finish
	p.wt.Wait()
}

// Pool manages a fixed number of worker goroutines consuming jobs from a shared queue.
// The pool is intentionally untyped to allow heterogeneous jobs to be executed.
type Pool struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wt         sync.WaitGroup
	closeOnce  sync.Once
	jobChannel chan Job
}

// NewPool creates a worker pool with the specified number of workers and job queue size.
// Workers start immediately and block waiting for jobs.
func NewPool(workerCount int, jobQueueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		ctx:        ctx,
		cancel:     cancel,
		wt:         sync.WaitGroup{},
		jobChannel: make(chan Job, jobQueueSize),
	}

	for i := range workerCount {
		worker := &worker{
			id:         i + 1,
			jobChannel: p.jobChannel,
		}
		p.wt.Go(worker.Start)
	}
	return p
}

// NewJob wraps a function without a return value into a Job.
func NewJob[PoolArgs any](fn Func[PoolArgs], args PoolArgs) Job {
	return &job[PoolArgs]{
		args: args,
		fn:   fn,
	}
}

// NewJobWithResult wraps a function with a return value into a Job.
// The result channel is owned by the caller and may be shared across jobs.
func NewJobWithResult[PoolArgs, PoolResult any](fn FuncWithResult[PoolArgs, PoolResult], args PoolArgs, resultChan chan PoolResult) Job {
	return &jobWithResult[PoolArgs, PoolResult]{
		args:       args,
		fn:         fn,
		resultChan: resultChan,
	}
}
