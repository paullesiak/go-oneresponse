package oneresponse

import (
	"context"
	"errors"
	"sync/atomic"
)

// OperationWithData is a generic function type that will allow some function to return a response and an error
type OperationWithData[T any] func(context.Context) (T, error)

// ErrNoOperations indicates no operations were provided.
var ErrNoOperations = errors.New("oneresponse: no operations")

// Serial will call multiple functions passed in with the same signature and return the first one that gives a
// non-error response in the order passed
func Serial[T any](ctx context.Context, operation []OperationWithData[T]) (T, error) {
	if len(operation) == 0 {
		var z T
		return z, ErrNoOperations
	}
	var errs []error
	var res T
	for _, op := range operation {
		var err error
		res, err = op(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return res, nil
	}
	return res, errors.Join(errs...)
}

// Parallel will call multiple functions passed in with the same signature in parallel and return the value for the
// first one that returns a non-error response. If no function returns successfully, a joined list of errors will be
// returned.
func Parallel[T any](ctx context.Context, operation []OperationWithData[T]) (T, error) {
	if len(operation) == 0 {
		var z T
		return z, ErrNoOperations
	}
	var errs []error
	var result T
	var success atomic.Bool
	resCh := make(chan T, 1)
	errCh := make(chan error, len(operation))
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := range operation {
		go func(o OperationWithData[T]) {
			res, err := o(subCtx)
			if err != nil {
				// if another operation already succeeded, skip reporting errors.
				if success.Load() {
					return
				}
				errCh <- err
				return
			}
			// atomically check and set the success flag to ensure only the first successful operation sends its result.
			if success.CompareAndSwap(false, true) {
				resCh <- res
				// proactively cancel remaining operations to encourage prompt shutdown.
				cancel()
			}
		}(operation[i])
	}
	for i := 0; i < len(operation); i++ {
		select {
		case result = <-resCh:
			return result, nil
		case err := <-errCh:
			errs = append(errs, err)
		}
	}
	return result, errors.Join(errs...)
}
