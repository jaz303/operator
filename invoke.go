package operator

import (
	"context"
	"errors"
	"fmt"
)

var ErrRecovered = errors.New("operation recovered from panic")

// InvokeTx() executes the supplied operation with the given input parameters.
//
// The supplied *Hub is used as a transaction provider and event dispatcher.
//
// Returns the operation's output on success, or error on failure.
func Invoke[Tx Transaction, I any, O any](ctx context.Context, hub *Hub[Tx], op Operation[Tx, I, O], input *I) (*O, error) {
	opCtx := hub.BeginOperation(ctx)

	output, err := invokeWithRecover(func() (*O, error) {
		return op(opCtx, input)
	})

	if err != nil {
		opCtx.rollback()
		return nil, err
	} else if err := opCtx.commit(); err != nil {
		return nil, fmt.Errorf("commit operation failed (%s)", err)
	}

	return output, nil
}

// InvokeTx() begins a transaction then executes the supplied operation with the
// given input parameters. Prefer InvokeTx() over Invoke() to reduce boilerplate
// if the operation is guaranteed to use a transaction.
//
// The supplied *Hub is used as a transaction provider and event dispatcher.
//
// Returns the operation's output on success, or error on failure.
func InvokeTx[Tx Transaction, I any, O any](ctx context.Context, hub *Hub[Tx], op TxOperation[Tx, I, O], input *I) (*O, error) {
	opCtx := hub.BeginOperation(ctx)

	tx, err := opCtx.Tx()
	if err != nil {
		return nil, err
	}

	output, err := invokeWithRecover(func() (*O, error) {
		return op(opCtx, tx, input)
	})

	if err != nil {
		opCtx.rollback()
		return nil, err
	} else if err := opCtx.commit(); err != nil {
		return nil, fmt.Errorf("commit operation failed (%s)", err)
	}

	return output, nil
}

func invokeWithRecover[O any](fn func() (*O, error)) (out *O, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("%w: %w", ErrRecovered, e)
			} else {
				err = fmt.Errorf("%w: %v", ErrRecovered, r)
			}
		}
	}()
	out, err = fn()
	return
}
