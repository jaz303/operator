package operator

import (
	"context"
)

// TODO: per-operation cache?
// TODO: logging/tracing functionality

// TODO: do we need an option to dispatch an event immediately?
// TODO: should we support events that don't receive an OpContext?
//       (these would be fired after commit, and would not return errors)

const (
	stateActive = iota
	stateDispatchEvents
	stateInvokeAfter
	stateSuccess
	stateFailed
	stateRolledback
)

// OpContext represents the context of in-process operation including
// its current state, associated Hub, and active transaction (if any).
//
// An OpContext is responsible for managing the lifecycle of its associated
// operation, including commit/rollback handling, event dispatch, and
// AfterFunc invocation. This functionality is not exposed publicly, instead,
// coordination is delegated to Invoke().
//
// OpContext wraps context.Context so can be passed to any method that
// expects one of these.
type OpContext[T Transaction] struct {
	context.Context

	hub              *Hub[T]
	beginTransaction TransactionProvider[T]

	state int

	activeTx T
	events   []Event
	after    []AfterFunc[T]
}

// Return the operation's transaction, creating a new transaction if not
// already started.
func (o *OpContext[T]) Tx() (T, error) {
	var zero T
	if !o.isTransactionActive() {
		tx, err := o.beginTransaction(o.Context)
		if err != nil {
			return zero, err
		}
		o.activeTx = tx
	}
	return o.activeTx, nil
}

// Register an event to be dispatched upon completion of the operation.
func (o *OpContext[T]) Emit(evt Event) error {
	if o.state <= stateDispatchEvents {
		return ErrInvalidState
	}
	o.events = append(o.events, evt)
	return nil
}

// Register a function to be invoked upon completion of the operation.
// The callback is invoked after the transaction (if any) is committed.
// After callbacks can be registered by the main operation, as well as
// any triggered event handlers.
func (o *OpContext[T]) AfterFunc(fn AfterFunc[T]) error {
	if o.state != stateActive && o.state != stateDispatchEvents {
		return ErrInvalidState
	}
	o.after = append(o.after, fn)
	return nil
}

func (o *OpContext[T]) commit() error {
	if o.state != stateActive {
		return ErrInvalidState
	}

	o.state = stateDispatchEvents
	if err := o.dispatchEvents(); err != nil {
		o.state = stateFailed
		if o.isTransactionActive() {
			_ = o.activeTx.Rollback(o.Context)
			// TODO: return appropriate error
		}
		return err
	}

	if o.isTransactionActive() {
		txErr := o.activeTx.Commit(o.Context)
		if txErr != nil {
			o.state = stateFailed
			return txErr
		}
	}

	o.state = stateInvokeAfter
	o.invokeAfterFuncs()

	o.state = stateSuccess

	return nil
}

func (o *OpContext[T]) rollback() error {
	if o.state != stateActive {
		return ErrInvalidState
	}

	o.state = stateRolledback

	if o.isTransactionActive() {
		return o.activeTx.Rollback(o.Context)
	}

	return nil
}

func (o *OpContext[T]) invokeAfterFuncs() {
	for _, fn := range o.after {
		fn(o)
	}
}

func (o *OpContext[T]) dispatchEvents() error {
	for len(o.events) > 0 {
		evt := o.events[0]
		o.events = o.events[1:]
		if err := o.hub.dispatchEvent(o, evt); err != nil {
			return err
		}
	}
	return nil
}

func (o *OpContext[T]) isTransactionActive() bool {
	var zero T
	return o.activeTx != zero
}
