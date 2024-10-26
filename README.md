# UnboundedQueue

`UnboundedQueue` is a Go library that provides a generic, unbounded queue with configurable options for handling task dispatch and processing. It is designed to process tasks in a FIFO order using a custom `TaskHandler` function, with options for blocking and non-blocking behavior when the queue reaches a specified length.

## Features

- **Generic Task Support**: `UnboundedQueue` can handle tasks of any type.
- **Configurable Blocking Behavior**: Choose between blocking and non-blocking behavior when the queue reaches its maximum length.
- **Graceful Shutdown**: Cancel the queue's context to prevent new tasks and process any remaining tasks before shutting down.
- **Wait for Task Completion**: Use `Wait` to block until all tasks in the queue have been processed.

## Installation

To install the package, use:

```bash
go get -u github.com/uptimepeer/unboundedqueue
```

## Usage

### Importing the Library

```go
import "github.com/uptimepeer/unboundedqueue"
```

### Basic Example

To use the `UnboundedQueue`, you need to define a task handler function, create a new queue, and start dispatching tasks to it. Below is a simple example of using `UnboundedQueue` to handle integer tasks:

```go
package main

import (
	"context"
	"fmt"
	"time"
	"github.com/uptimepeer/synchronizedbuffer"
)

func main() {
	// Create a context to manage the queue's lifetime.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled on exit.

	// Define a task handler function for integer tasks.
	taskHandler := func(task int) {
		fmt.Printf("Processing task: %d\n", task)
		time.Sleep(500 * time.Millisecond) // Simulate task processing time
	}

	// Create an UnboundedQueue with the task handler and default options.
	queue := unboundedqueue.NewUnboundedQueue(ctx, taskHandler)

	// Dispatch tasks to the queue.
	for i := 1; i <= 10; i++ {
		err := queue.Dispatch(i)
		if err != nil {
			fmt.Printf("Failed to dispatch task %d: %v\n", i, err)
		}
	}

	// Close and wait for all tasks to be processed before exiting.
    queue.Close()
	queue.Wait()
	fmt.Println("All tasks processed.")
}
```

> **Note**: The queue can be shut down by either canceling the `ctx` or calling `Close()` on the queue. If you don’t need to operate with a context, you may safely pass `nil` when creating the queue using `NewUnboundedQueue`.

### Configuration Options

The `UnboundedQueue` provides several configuration options to customize its behavior. These options can be set using `OptionFunc` functions during initialization:

#### WithMaxLength

Limits the number of tasks in the queue. If the queue length is exceeded:
- **Blocking Mode**: The `Dispatch` method will wait until there is room in the queue.
- **Non-Blocking Mode**: The `Dispatch` method will return an error if the queue is full.

```go
queue := unboundedqueue.NewUnboundedQueue(ctx, taskHandler, unboundedqueue.WithMaxLength(100))
```

#### WithBlocking

Controls the behavior when the queue reaches the maximum length.
- `WithBlocking(true)` (default): `Dispatch` will block until there is room in the queue.
- `WithBlocking(false)`: `Dispatch` will return an error (`ErrQueueFull`) if the queue is full.

```go
queue := unboundedqueue.NewUnboundedQueue(ctx, taskHandler, unboundedqueue.WithBlocking(true))
```

### Graceful Shutdown

To gracefully shut down the queue, cancel the queue’s context or call `Close()` directly. This will stop accepting new tasks and allow all queued tasks to complete.

> **Note**: If the queue is closed by calling the `cancel()` function on the context, there is no need to call `Close()` separately.

```go
// Cancel the queue's context to stop accepting new tasks.
queue.Close()

// Wait for all ongoing tasks to complete before exiting.
queue.Wait()
```

### Error Handling

`Dispatch` may return the following errors:
- `ErrQueueFull`: Returned when the queue reaches its maximum length and is in non-blocking mode.
- `ErrQueueClosed`: Returned when attempting to add a task to a queue that has been closed.

## API Reference

The following are the public methods available in `UnboundedQueue`:

### NewUnboundedQueue

Creates a new `UnboundedQueue` with a specified `TaskHandler` and optional configuration options.

```go
ctx := context.Background()
taskHandler := func(task int) { fmt.Println(task) }
queue := unboundedqueue.NewUnboundedQueue(ctx, taskHandler, unboundedqueue.WithMaxLength(10), unboundedqueue.WithBlocking(true))
```

### Dispatch

Adds a task to the queue for processing. If the queue is at capacity, `Dispatch` will either block or return an error depending on the configuration.

```go
err := queue.Dispatch(16)
if err != nil {
	fmt.Println("Error dispatching task: ", err)
}
```

### Close

Closes the queue, preventing any new tasks from being added. This is an alternative to canceling the context. 

```go
queue.Close()
```

### Wait

Blocks until all tasks in the queue have been processed.

```go
queue.Wait()
```

### QueueLength

Returns the current length of the queue (number of tasks waiting to be processed).

```go
length := queue.QueueLength()
fmt.Println("Current queue length: ", length)
```

## Contributing

Feel free to open issues or submit pull requests with improvements or bug fixes.

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
