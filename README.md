# Worker Pool (pool)

The pool package provides a generic, untyped worker pool for Go that can execute heterogeneous jobs concurrently.
Jobs may accept arbitrary argument types and may optionally return results.

The pool itself operates on a common Job interface, allowing different kinds of work to coexist in the same pool while keeping worker management simple and efficient.

This design is particularly useful when:
- You need a fixed-size worker pool
- Jobs vary in type and behavior
- You want explicit control over result handling and synchronization

## Features
- **Generic Jobs**: Utilize Go generics to pass any type of arguments and receive any type of results.
- **Heterogeneous Execution**: A single pool can handle different types of jobs simultaneously.
- **Context Support**: `AddJob` respects `context.Context` for cancellation and timeouts when the queue is full.
- **Concurrency Control**: Define a fixed number of workers and job queue size to manage resources effectively.

## Installation

```go
go get github.com/Kenasvarghese/Worker-pool
```

## Design Overview
- The pool manages a fixed number of worker goroutines.
- Workers consume jobs from a shared channel.
- Jobs implement a simple Job interface:

```go
type Job interface {
    Run()
}
```
- Generic helper functions wrap typed functions into untyped Job values.
- The pool itself is intentionally untyped, enabling mixed workloads.

## Usage

### 1. Initialize the Pool

Create a pool with a specific number of workers and a buffered job channel size.

```go
package main

import (
	"github.com/Kenasvarghese/Worker-pool/pool"
)

func main() {
	// Create a pool with 5 workers and a queue capacity of 10
	p := pool.NewPool(5, 10)
	defer p.Close()
    
    // ...
}
```

### 2. Creating and Adding Jobs

You can create simple fire-and-forget jobs or jobs that return a result.

#### Fire-and-Forget Job

Use `NewJob` for tasks that don't need to return a value to the caller.

```go
import (
	"context"
	"fmt"
	"github.com/Kenasvarghese/Worker-pool/pool"
)

func main() {
	p := pool.NewPool(5, 10)
    
	// Define a job function
	jobFunc := func(name string) error {
		fmt.Printf("Hello, %s!\n", name)
		return nil
	}

	// Create the job wrapper
	job := pool.NewJob(jobFunc, "World")

	// Submit the job
	p.AddJob(context.Background(), job)
}
```

#### Job with Result

Use `NewJobWithResult` when you need to capture the output of a task.

```go
func main() {
	p := pool.NewPool(5, 10)

	// Channel to receive the result
	resultChan := make(chan int, 1)

	// Define a job function that returns a value
	squareFunc := func(n int) (int, error) {
		return n * n, nil
	}

	// Create a job providing the function, arguments, and result channel
	job := pool.NewJobWithResult(squareFunc, 10, resultChan)

	p.AddJob(context.Background(), job)

	// Wait for the result
	result := <-resultChan
	fmt.Printf("Result: %d\n", result)
}
```

## Concurrency & Channel Ownership Rules

Important design rules
- Result channels are owned by the caller
- Jobs must not close shared result channels
- Callers are responsible for:
- Synchronization
- Channel buffering
- Channel closure (when appropriate)

This design avoids hidden lifecycle coupling and allows:
- Fan-in patterns
- Shared result aggregation
- External coordination with sync.WaitGroup, contexts, etc.

## API Overview

```go
NewPool(workerCount int, jobQueueSize int) *Pool
```
Creates a new worker pool. Workers start immediately.

```go
NewJob[T any](fn Func[T], args T) Job
```
Creates a job that executes `fn(args)`. Use this for side-effect only operations.

```go
NewJobWithResult[T, R any](fn FuncWithResult[T, R], args T, resultChan chan R) Job
```
Creates a job that executes `fn(args)` and sends the result to `resultChan`.

```go
(p *Pool) AddJob(ctx context.Context, job Job)
```
Adds a job to the queue. Blocks if queue is full or context is cancelled.

 ```go
 (p *Pool) Close()
 ```
Closes the internal job channel. Workers will exit after processing remaining jobs.
