package operator

import (
	"context"
	"errors"
)

var (
	ErrInvalidState               = errors.New("invalid state")
	ErrEventHandlerCoercionFailed = errors.New("failed to create event handler")
)

// Operation represents a single operation with defined input/output parameters.
type Operation[Tx Transaction, I any, O any] func(ctx *OpContext[Tx], input *I) (*O, error)

// TxOperation represents a single operation with an implied transaction, and defined input/output parameters.
// Use TxOperation to reduce boilerplate if your operation is guaranteed to start a transaction.
type TxOperation[Tx Transaction, I any, O any] func(ctx *OpContext[Tx], tx Tx, input *I) (*O, error)

// AfterFunc is a function that runs after an operation has successfully completed.
type AfterFunc[Tx Transaction] func(*OpContext[Tx])

// TransactionProvider is a transaction factory
type TransactionProvider[Tx Transaction] func(context.Context) (Tx, error)

type Transaction interface {
	comparable

	Commit(context.Context) error
	Rollback(context.Context) error
}
