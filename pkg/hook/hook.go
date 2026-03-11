// Package hook defines the public interface for http-pull hooks.
//
// Hooks are actions executed after an HTTP pull completes. Third-party
// implementations can satisfy the Hook interface and be integrated into
// the http-pull pipeline.
package hook

import "context"

// Trigger describes when a hook should fire.
type Trigger int

const (
	// TriggerSuccess fires the hook when the pull succeeds.
	TriggerSuccess Trigger = iota
	// TriggerFailure fires the hook when the pull fails.
	TriggerFailure
)

// Result carries information about a completed pull operation.
type Result struct {
	// Success indicates whether the pull completed without error.
	Success bool
	// FilePath is the local path where the downloaded content was stored.
	FilePath string
	// Err holds the error if the pull failed, or nil on success.
	Err error
}

// Hook is the interface that all hook implementations must satisfy.
//
// Execute is called after a pull operation completes. The provided context
// may carry a cancellation signal for graceful shutdown. Implementations
// should respect context cancellation and return promptly when cancelled.
type Hook interface {
	// Execute runs the hook action. The result describes the preceding pull.
	Execute(ctx context.Context, result Result) error
}
