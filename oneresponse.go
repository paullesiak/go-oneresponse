package oneresponse

import (
	"context"
	"errors"
	"sync/atomic"
)

// OperationWithData is a generic function type that will allow some function to return a response and an error
type OperationWithData[T any] func(context.Context) (T, error)

// Serial will call multiple functions passed in with the same signature, and return the first one that gives a
// non-error response in the order passed
func Serial[T any](ctx context.Context, operation []OperationWithData[T]) (T, error) {
	var errs []error
	var res T
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, op := range operation {
		var err error
		res, err = op(subCtx)
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
	var errs []error
	var result T
	var success atomic.Bool
	resCh := make(chan T, len(operation))
	errCh := make(chan error, len(operation))
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := range operation {
		go func(index int, o OperationWithData[T]) {
			res, err := o(subCtx)
			if err != nil {
				errCh <- err
				return
			}
			success.Store(true)
			resCh <- res
		}(i, operation[i])
	}
consumeLoop:
	for {
		select {
		case result = <-resCh:
			cancel()
			break consumeLoop
		case err := <-errCh:
			errs = append(errs, err)
			if len(errs) == len(operation) {
				break consumeLoop
			}
		}
	}
	if success.Load() {
		return result, nil
	}
	return result, errors.Join(errs...)
}
