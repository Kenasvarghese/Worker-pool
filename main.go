package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenasvarghese/Worker-pool/pool"
)

func main() {
	// Create a pool with 5 workers and a job queue size of 10
	p := pool.NewPool(5, 10)
	defer p.Close()

	// -------------------------
	// Job WITHOUT a result
	// -------------------------
	noResultJob := pool.NewJob(func(n int) error {
		fmt.Println("no-result job ran with:", n)
		return nil
	}, 1)

	p.AddJob(context.Background(), noResultJob)

	// -------------------------
	// Job WITH a result
	// -------------------------
	resultChan := make(chan int, 1)

	resultJob := pool.NewJobWithResult(
		func(n int) (int, error) {
			fmt.Println("result job ran with:", n)
			return n * 2, nil
		},
		10,
		resultChan,
	)

	p.AddJob(context.Background(), resultJob)

	// Receive the result
	result := <-resultChan
	fmt.Println("received result:", result)

	// Allow time for workers to finish before main exits
	time.Sleep(500 * time.Millisecond)
}
