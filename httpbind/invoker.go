package httpbind

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jaz303/operator"
	"github.com/jaz303/operator/operr"
)

// Bind() creates an an Invoker binding the operation to an HTTP endpoint
// The returned Invoker can be further customised before finally calling Go()
func Bind[Tx operator.Transaction, Ext any, I any, O any](
	hub *operator.Hub[Tx, Ext],
	op func(*operator.OpContext[Tx, Ext], *I) (*O, error),
) *Invoker[Tx, Ext, I, O] {
	return &Invoker[Tx, Ext, I, O]{
		hub: hub,
		op:  op,

		ctx:         func(r *http.Request) context.Context { return context.Background() },
		errorMapper: operr.DefaultErrorMapper,
	}
}

// BindTx() creates an an Invoker binding the operation to an HTTP endpoint
// The returned Invoker can be further customised before finally calling Go()
func BindTx[Tx operator.Transaction, Ext any, I any, O any](
	hub *operator.Hub[Tx, Ext],
	op func(*operator.OpContext[Tx, Ext], Tx, *I) (*O, error),
) *Invoker[Tx, Ext, I, O] {
	return &Invoker[Tx, Ext, I, O]{
		hub:  hub,
		txOp: op,

		ctx:         func(r *http.Request) context.Context { return context.Background() },
		errorMapper: operr.DefaultErrorMapper,
	}
}

// Invoker acts as a configuration point when binding operations to HTTP endpoints.
// Use its With* functions to customise input, output, and error behaviour, then call
// Go() to invoke the operation.
type Invoker[Tx operator.Transaction, Ext any, I any, O any] struct {
	hub  *operator.Hub[Tx, Ext]
	op   func(*operator.OpContext[Tx, Ext], *I) (*O, error)
	txOp func(*operator.OpContext[Tx, Ext], Tx, *I) (*O, error)

	ctx          func(r *http.Request) context.Context
	inputMapper  func(r *http.Request) (*I, error)
	outputMapper func(w http.ResponseWriter, o *O)
	errorMapper  func(w http.ResponseWriter, err error)
}

// WithContext() sets a static context for the operation
func (i *Invoker[Tx, Ext, I, O]) WithContext(ctx context.Context) *Invoker[Tx, Ext, I, O] {
	i.ctx = func(r *http.Request) context.Context { return ctx }
	return i
}

// WithContextFn() sets fn as a context factory the operation.
// This can be used, for an example, to use the HTTP request's context as the operation context.
// (in most cases this is undesirable, however).
func (i *Invoker[Tx, Ext, I, O]) WithContextFunc(fn func(*http.Request) context.Context) *Invoker[Tx, Ext, I, O] {
	i.ctx = fn
	return i
}

// WithInputMapper() registers the binding's input mapper
func (i *Invoker[Tx, Ext, I, O]) WithInputMapper(fn func(*http.Request) (*I, error)) *Invoker[Tx, Ext, I, O] {
	i.inputMapper = fn
	return i
}

// WithOutputMapper() registers the binding's output mapper
func (i *Invoker[Tx, Ext, I, O]) WithOutputMapper(fn func(w http.ResponseWriter, o *O)) *Invoker[Tx, Ext, I, O] {
	i.outputMapper = fn
	return i
}

// WithJSONOutputFunc() is a shortcut method for the common pattern of transforming an operation's output into JSON
func (i *Invoker[Tx, Ext, I, O]) WithJSONOutputFunc(fn func(w http.ResponseWriter, o *O)) *Invoker[Tx, Ext, I, O] {
	i.outputMapper = func(w http.ResponseWriter, o *O) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, o)
	}
	return i
}

// WithJSONOutput() is a shortcut method for the common pattern of writing an operation's output directly as JSON
func (i *Invoker[Tx, Ext, I, O]) WithJSONOutput(fn func(w http.ResponseWriter, o *O) any) *Invoker[Tx, Ext, I, O] {
	i.outputMapper = func(w http.ResponseWriter, o *O) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fn(w, o))
	}
	return i
}

// Register an error mapper for writing an error to the HTTP response.
//
// The error provided to the callback wraps both the source error, and one of either
// operr.ErrInputMappingFailed or operr.ErrOperationFailed, to indicate in which phase
// the error occurred.
//
// Since you will likely use the same error mapper for every operation, to avoid
// registering the mapper each time, it is common to wrap Bind() and BindTx() to attach
// your preferred handler automatically.
func (i *Invoker[Tx, Ext, I, O]) WithErrorMapper(fn func(w http.ResponseWriter, err error)) *Invoker[Tx, Ext, I, O] {
	i.errorMapper = fn
	return i
}

// Invoke the bound operation in the context of the supplied HTTP request
func (i *Invoker[Tx, Ext, I, O]) Go(w http.ResponseWriter, r *http.Request) {
	input, err := i.getInputMapper()(r)
	if err != nil {
		i.errorMapper(w, fmt.Errorf("%w: %w", operr.ErrInputMappingFailed, err))
		return
	}

	var output *O
	if i.txOp != nil {
		output, err = operator.InvokeTx(i.getContext(r), i.hub, i.txOp, input)
	} else {
		output, err = operator.Invoke(i.getContext(r), i.hub, i.op, input)
	}

	if err != nil {
		i.errorMapper(w, fmt.Errorf("%w: %w", operr.ErrOperationFailed, err))
		return
	}

	i.getOutputMapper()(w, output)
}

func (i *Invoker[Tx, Ext, I, O]) getContext(r *http.Request) context.Context {
	return i.ctx(r)
}

func (i *Invoker[Tx, Ext, I, O]) getInputMapper() func(r *http.Request) (*I, error) {
	if i.inputMapper == nil {
		return Zero[I]
	}
	return i.inputMapper
}

func (i *Invoker[Tx, Ext, I, O]) getOutputMapper() func(http.ResponseWriter, *O) {
	if i.outputMapper == nil {
		return WriteJSON[O]
	}
	return i.outputMapper
}
