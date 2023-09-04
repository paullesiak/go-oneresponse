package oneresponse

import (
	"errors"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

/*
oneresponse will call multiple functions with the same signature, and return the first one that returns a non-error
response, otherwise will return a nil value with a concatenated list of errors. any non-initial response will be dropped
*/

// OperationWithData is a generic function type that will allow some function to return a response and an error
type OperationWithData[T any] func() (T, error)

// Serial will call multiple functions passed in with the same signature, and return the first one that gives a positive
// reply in a serial order
func Serial[T any](operation []OperationWithData[T]) (T, error) {
	var errs []error
	success := atomic.Bool{}
	var res T
	for _, op := range operation {
		var err error
		res, err = op()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		success.Store(true)
		return res, nil
	}
	if success.Load() {
		return res, nil
	}
	return res, errors.Join(errs...)
}

// Parallel will call multiple functions passed in with the same signature, and return the first one that gives
// a positive reply in a sequential order
func Parallel[T any](operation []OperationWithData[T]) (T, error) {
	var errs []error
	var result T
	var success atomic.Bool
	resCh := make(chan T, len(operation))
	errCh := make(chan error, len(operation))
	log.Info().Msg("Starting operations")
	for i := range operation {
		go func(index int, o OperationWithData[T]) {
			res, err := o()
			if err != nil {
				log.Error().Err(err).Msgf("operation %d failed", index)
				errCh <- err
				return
			}
			success.Store(true)
			log.Info().Interface("result", res).Msgf("operation %d succeeded", index)
			resCh <- res
		}(i, operation[i])
	}
	log.Info().Msg("Waiting for responses")
consumeLoop:
	for {
		select {
		case result = <-resCh:
			log.Info().Msg("got a response")
			break consumeLoop
		case err := <-errCh:
			errs = append(errs, err)
			log.Error().Msg("got an error")
			if len(errs) == len(operation) {
				log.Error().Msg("errors for all operations")
				break consumeLoop
			}
		}
	}
	if success.Load() {
		return result, nil
	}
	return result, errors.Join(errs...)
}
