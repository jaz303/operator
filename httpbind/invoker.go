package httpbind

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jaz303/operator"
	"github.com/jaz303/operator/operr"
)

func Bind[Tx operator.Transaction, I any, O any](
	hub *operator.Hub[Tx],
	op func(*operator.OpContext[Tx], *I) (*O, error),
) *Invoker[Tx, I, O] {
	return &Invoker[Tx, I, O]{
		hub:         hub,
		op:          op,
		errorMapper: operr.DefaultErrorMapper,
	}
}

func BindTx[Tx operator.Transaction, I any, O any](
	hub *operator.Hub[Tx],
	op func(*operator.OpContext[Tx], Tx, *I) (*O, error),
) *Invoker[Tx, I, O] {
	return &Invoker[Tx, I, O]{
		hub:         hub,
		txOp:        op,
		errorMapper: operr.DefaultErrorMapper,
	}
}

type Invoker[Tx operator.Transaction, I any, O any] struct {
	hub  *operator.Hub[Tx]
	op   func(*operator.OpContext[Tx], *I) (*O, error)
	txOp func(*operator.OpContext[Tx], Tx, *I) (*O, error)

	ctx          func(r *http.Request) context.Context
	inputMapper  func(r *http.Request) (*I, error)
	outputMapper func(w http.ResponseWriter, o *O)
	errorMapper  func(w http.ResponseWriter, err error)
}

//
// Context

func (i *Invoker[Tx, I, O]) WithContext(ctx context.Context) *Invoker[Tx, I, O] {
	i.ctx = func(r *http.Request) context.Context {
		return ctx
	}
	return i
}

func (i *Invoker[Tx, I, O]) WithContextFunc(fn func(*http.Request) context.Context) *Invoker[Tx, I, O] {
	i.ctx = fn
	return i
}

//
// Input mapper

func (i *Invoker[Tx, I, O]) WithInputMapper(fn func(*http.Request) (*I, error)) *Invoker[Tx, I, O] {
	i.inputMapper = fn
	return i
}

//
// Output mapper

func (i *Invoker[Tx, I, O]) WithOutputMapper(fn func(w http.ResponseWriter, o *O)) *Invoker[Tx, I, O] {
	i.outputMapper = fn
	return i
}

func (i *Invoker[Tx, I, O]) WithJSONOutputFunc(fn func(w http.ResponseWriter, o *O)) *Invoker[Tx, I, O] {
	i.outputMapper = func(w http.ResponseWriter, o *O) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, o)
	}
	return i
}

func (i *Invoker[Tx, I, O]) WithJSONOutput(fn func(w http.ResponseWriter, o *O) any) *Invoker[Tx, I, O] {
	i.outputMapper = func(w http.ResponseWriter, o *O) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fn(w, o))
	}
	return i
}

//
// Error mapper

func (i *Invoker[Tx, I, O]) WithErrorMapper(fn func(w http.ResponseWriter, err error)) *Invoker[Tx, I, O] {
	i.errorMapper = fn
	return i
}

//
// Invoke

func (i *Invoker[Tx, I, O]) Go(w http.ResponseWriter, r *http.Request) {
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

func (i *Invoker[Tx, I, O]) getContext(r *http.Request) context.Context {
	if i.ctx == nil {
		return r.Context()
	}
	return i.ctx(r)
}

func (i *Invoker[Tx, I, O]) getInputMapper() func(r *http.Request) (*I, error) {
	if i.inputMapper == nil {
		return Zero[I]
	}
	return i.inputMapper
}

func (i *Invoker[Tx, I, O]) getOutputMapper() func(http.ResponseWriter, *O) {
	if i.outputMapper == nil {
		return WriteJSON[O]
	}
	return i.outputMapper
}
