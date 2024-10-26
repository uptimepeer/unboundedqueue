package unboundedqueue

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrQueueFull is returned by Dispatch when the queue has reached its maximum length and is non-blocking.
	ErrQueueFull = errors.New("cannot dispatch task: queue is full")

	// ErrQueueClosed is returned by Dispatch when attempting to add a task to a closed queue.
	ErrQueueClosed = errors.New("cannot dispatch task: queue is closed")
)

// TaskHandler is a function type that defines how each task will be handled by the queue.
type TaskHandler[T any] func(T)

// UnboundedQueue is a generic unbounded queue with configurable options.
// Tasks are processed in FIFO order by the provided TaskHandler function.
type UnboundedQueue[T any] struct {
	buffer      []T                // Holds the enqueued tasks.
	taskHandler TaskHandler[T]     // Function to handle each task.
	wg          sync.WaitGroup     // WaitGroup to track task completion.
	mu          sync.Mutex         // Mutex to protect access to buffer.
	cond        *sync.Cond         // Condition variable for coordinating goroutines.
	ctx         context.Context    // Context of queue.
	cancelFunc  context.CancelFunc // Function to cancel the queue's context.
	options     QueueOptions       // Configuration options for the queue.
}

// QueueOptions holds configuration settings for the UnboundedQueue.
type QueueOptions struct {
	maxQueueLength int  // Maximum length of the queue before blocking or returning an error.
	blockingFlag   bool // Indicates if Dispatch should block when the queue is full.
}

// OptionFunc defines a function type for configuring QueueOptions.
type OptionFunc func(*QueueOptions)

// WithMaxLength sets a maximum length for the queue, allowing Dispatch to block or error out if exceeded.
func WithMaxLength(length int) OptionFunc {
	return func(o *QueueOptions) {
		o.maxQueueLength = length
	}
}

// WithBlocking determines whether Dispatch should block if the queue is full.
func WithBlocking(blocking bool) OptionFunc {
	return func(o *QueueOptions) {
		o.blockingFlag = blocking
	}
}

// NewUnboundedQueue initializes a new UnboundedQueue with the provided options and starts a background worker to process tasks.
func NewUnboundedQueue[T any](ctx context.Context, taskHandler TaskHandler[T], opts ...OptionFunc) *UnboundedQueue[T] {
	// Default queue options
	options := QueueOptions{
		maxQueueLength: 0,    // No limit on queue length by default.
		blockingFlag:   true, // Dispatch will block if the queue is full.
	}

	// Apply option functions to customize options
	for _, opt := range opts {
		opt(&options)
	}

	// Ensure the context is not nil
	if ctx == nil {
		ctx = context.Background()
	}

	// Create a cancellable context for the queue
	ctx, cancelFunc := context.WithCancel(ctx)

	// Initialize the queue
	q := &UnboundedQueue[T]{
		buffer:      make([]T, 0),
		taskHandler: taskHandler,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		options:     options,
	}

	// Create a new condition variable linked to the mutex
	q.cond = sync.NewCond(&q.mu)

	// Start the background task processing goroutine
	go q.run(ctx)

	return q
}

// Dispatch enqueues a task for processing.
// It returns ErrQueueFull if the queue is full in non-blocking mode,
// or ErrQueueClosed if the context was canceled or queue closed.
func (q *UnboundedQueue[T]) Dispatch(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if the queue context has been canceled
	if q.ctx.Err() != nil {
		return ErrQueueClosed
	}

	// If the queue has a maximum length and is full, handle blocking or non-blocking behavior
	if q.options.maxQueueLength > 0 && len(q.buffer) >= q.options.maxQueueLength {
		if q.options.blockingFlag {
			// Block until space becomes available or context is canceled
			for len(q.buffer) >= q.options.maxQueueLength {
				q.cond.Wait()
				if q.ctx.Err() != nil {
					return ErrQueueClosed
				}
			}
		} else {
			return ErrQueueFull
		}
	}

	// Enqueue the task
	q.buffer = append(q.buffer, item)
	q.wg.Add(1) // Increment WaitGroup counter to track this task's completion

	// Signal the condition variable to wake up the processing goroutine
	q.cond.Signal()

	return nil
}

// run processes tasks from the queue in a background goroutine.
// It runs until the queue context is canceled and all tasks are completed.
func (q *UnboundedQueue[T]) run(ctx context.Context) {
	for {
		q.mu.Lock()

		// Wait while the queue is empty and context has not been canceled
		for ctx.Err() == nil && len(q.buffer) == 0 {
			q.cond.Wait()
		}

		// Exit if the context is canceled and no tasks are left in the buffer
		if ctx.Err() != nil && len(q.buffer) == 0 {
			q.mu.Unlock()
			return
		}

		// Dequeue the next task
		task := q.buffer[0]
		q.buffer = q.buffer[1:]

		q.mu.Unlock()

		// Process the task outside of the locked section
		q.taskHandler(task)
		q.wg.Done() // Mark the task as done in the WaitGroup

		// Notify waiting goroutines that space is now available in the buffer
		q.mu.Lock()
		q.cond.Signal()
		q.mu.Unlock()
	}
}

// Close cancels the queue's context and broadcasts a signal to unblock any waiting goroutines.
func (q *UnboundedQueue[T]) Close() {
	q.cancelFunc() // Cancel the context to indicate closure

	q.mu.Lock()
	q.cond.Broadcast() // Wake up all waiting goroutines to let them exit
	q.mu.Unlock()
}

// Wait blocks until all tasks in the queue have been processed.
func (q *UnboundedQueue[T]) Wait() {
	q.wg.Wait() // Wait for all tasks tracked by the WaitGroup to complete
}

// QueueLength returns the current number of tasks in the queue.
func (q *UnboundedQueue[T]) QueueLength() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.buffer)
}
