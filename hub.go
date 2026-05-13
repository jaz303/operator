package operator

import (
	"context"
	"reflect"
	"sync"
	"time"
)

// A Hub is the central object through which operations are invoked, comprising
// a transaction provider, and a registry of event handlers.
//
// Once a Hub is configured, use the package-level Invoke() function to invoke
// operations.
type Hub[Tx Transaction, Ext any] struct {
	beginTransaction TransactionProvider[Tx]
	createExt        func() Ext
	eventHandlers    map[reflect.Type][]eventHandler[Tx, Ext]
}

// NewHub() returns a hub configured with a transaction provider.
func NewHub[Tx Transaction, Ext any](transactionProvider TransactionProvider[Tx], createExt func() Ext) *Hub[Tx, Ext] {
	return &Hub[Tx, Ext]{
		beginTransaction: transactionProvider,
		createExt:        createExt,
		eventHandlers:    map[reflect.Type][]eventHandler[Tx, Ext]{},
	}
}

// RegisterEventHandler() registers a handler to handle events whose
// type matches reflect.TypeOf(event).
//
// The event handler hnd must be a function conforming to one of the
// following signatures, wherein *OpContext[Tx] must be assignable to C
// (this includes context.Context), and the event type must be
// assignable to E:
//
// func(E)
// func(C, E)
// func(E) error
// func(C, E) error
//
// Event handlers are invoked *after* the operation has returned, but
// before the transaction is committed. If an event handler returns an
// error, the transaction aborts and is rolled back - this is by design;
// event handlers are not intended for "fire and forget" use - use AfterFunc()
// for that.
func (h *Hub[Tx, Ext]) RegisterEventHandler(event Event, hnd any) {
	ty := reflect.TypeOf(event)
	h.eventHandlers[ty] = append(h.eventHandlers[ty], makeEventHandler[Tx, Ext](ty, hnd))
}

// Begin a new operation and returns its context.
// User code will usually not call BeginOperation directly; use Invoke().
func (h *Hub[Tx, Ext]) BeginOperation(ctx context.Context) *OpContext[Tx, Ext] {
	return &OpContext[Tx, Ext]{
		Context: ctx,

		now: time.Now(),

		hub:              h,
		ext:              sync.OnceValue(h.createExt),
		beginTransaction: h.beginTransaction,
	}
}

func (h *Hub[Tx, Ext]) dispatchEvent(op *OpContext[Tx, Ext], evt Event) error {
	for _, hnd := range h.eventHandlers[reflect.TypeOf(evt)] {
		if err := hnd.Dispatch(op, evt); err != nil {
			return err
		}
	}
	return nil
}
