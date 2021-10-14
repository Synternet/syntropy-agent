// Stoppable/Startable context

package scontext

import (
	"context"
	"errors"
)

type StartStopContext struct {
	parentCtx, ctx context.Context
	cancel         context.CancelFunc
}

var (
	ErrRunning       = errors.New("already running")
	ErrStopped       = errors.New("not running")
	ErrParentStopped = errors.New("parent context stopped")
)

func New(ctx context.Context) StartStopContext {
	return StartStopContext{
		parentCtx: ctx,
	}
}

// Return parent context if not started and cancellable context if Start was
// previously called and was not cancelled using Stop.
// NOTE: This has to be used inside a mutex or other synchronization primitives
// if concurrent calls to Start/Stop are expected.
func (sc *StartStopContext) Context() context.Context {
	if sc.cancel == nil {
		return sc.parentCtx
	}
	return sc.ctx
}

// Create a cancellable context and return it.
// Will fail when either this context was already started previously or
// parent context was cancelled.
// NOTE: This has to be used inside a mutex or other synchronization primitives
// if concurrent calls are expected.
func (sc *StartStopContext) CreateContext() (context.Context, error) {
	if sc.cancel != nil {
		return nil, ErrRunning
	}

	select {
	case <-sc.parentCtx.Done():
		return nil, ErrParentStopped
	default:
	}

	sc.ctx, sc.cancel = context.WithCancel(sc.parentCtx)
	return sc.ctx, nil
}

// Cancel underlying context.
// Will fail if either the context wasn't started or parent context was stopped.
// NOTE: This has to be used inside a mutex or other synchronization primitives
// if concurrent calls are expected.
func (sc *StartStopContext) CancelContext() error {
	if sc.cancel == nil {
		return ErrStopped
	}

	defer func() { sc.cancel = nil }()
	select {
	case <-sc.ctx.Done():
		return ErrStopped
	default:
	}

	sc.cancel()
	return nil
}
