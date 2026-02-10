package echobind

import (
	"context"

	"github.com/jaz303/operator"
	"github.com/labstack/echo/v5"
)

// Bind creates an Invoker binding the operation to an Echo handler.
// The returned Invoker can be further customised before finally calling Go().
func Bind[Tx operator.Transaction, I any, O any](
	hub *operator.Hub[Tx],
	op func(*operator.OpContext[Tx], *I) (*O, error),
) *Invoker[Tx, I, O] {
	return &Invoker[Tx, I, O]{
		hub: hub,
		op:  op,

		ctx: func(c *echo.Context) context.Context { return context.Background() },
	}
}

// BindTx creates an Invoker binding the transactional operation to an Echo handler.
// The returned Invoker can be further customised before finally calling Go().
func BindTx[Tx operator.Transaction, I any, O any](
	hub *operator.Hub[Tx],
	op func(*operator.OpContext[Tx], Tx, *I) (*O, error),
) *Invoker[Tx, I, O] {
	return &Invoker[Tx, I, O]{
		hub:  hub,
		txOp: op,

		ctx: func(c *echo.Context) context.Context { return context.Background() },
	}
}

// Invoker acts as a configuration point when binding operations to Echo handlers.
// Use its With* functions to customise input and output behaviour, then call
// Go() to invoke the operation.
type Invoker[Tx operator.Transaction, I any, O any] struct {
	hub  *operator.Hub[Tx]
	op   func(*operator.OpContext[Tx], *I) (*O, error)
	txOp func(*operator.OpContext[Tx], Tx, *I) (*O, error)

	ctx          func(c *echo.Context) context.Context
	inputMapper  func(c *echo.Context) (*I, error)
	outputMapper func(c *echo.Context, o *O) error
}

// WithContext sets a static context for the operation
func (i *Invoker[Tx, I, O]) WithContext(ctx context.Context) *Invoker[Tx, I, O] {
	i.ctx = func(c *echo.Context) context.Context { return ctx }
	return i
}

// WithContextFunc sets fn as a context factory for the operation.
func (i *Invoker[Tx, I, O]) WithContextFunc(fn func(*echo.Context) context.Context) *Invoker[Tx, I, O] {
	i.ctx = fn
	return i
}

// WithInputMapper registers the binding's input mapper
func (i *Invoker[Tx, I, O]) WithInputMapper(fn func(*echo.Context) (*I, error)) *Invoker[Tx, I, O] {
	i.inputMapper = fn
	return i
}

// WithOutputMapper registers the binding's output mapper
func (i *Invoker[Tx, I, O]) WithOutputMapper(fn func(c *echo.Context, o *O) error) *Invoker[Tx, I, O] {
	i.outputMapper = fn
	return i
}

// WithJSONOutputFunc sets an output mapper that calls fn after setting the JSON content type
func (i *Invoker[Tx, I, O]) WithJSONOutputFunc(fn func(c *echo.Context, o *O) error) *Invoker[Tx, I, O] {
	i.outputMapper = fn
	return i
}

// WithJSONOutput sets an output mapper that writes the result of fn as JSON
func (i *Invoker[Tx, I, O]) WithJSONOutput(fn func(c *echo.Context, o *O) any) *Invoker[Tx, I, O] {
	i.outputMapper = func(c *echo.Context, o *O) error {
		return c.JSON(200, fn(c, o))
	}
	return i
}

// Go invokes the bound operation in the context of the supplied Echo request.
// Its signature matches echo.HandlerFunc.
func (i *Invoker[Tx, I, O]) Go(c *echo.Context) error {
	input, err := i.getInputMapper()(c)
	if err != nil {
		return err
	}

	var output *O
	if i.txOp != nil {
		output, err = operator.InvokeTx(i.getContext(c), i.hub, i.txOp, input)
	} else {
		output, err = operator.Invoke(i.getContext(c), i.hub, i.op, input)
	}

	if err != nil {
		return err
	}

	return i.getOutputMapper()(c, output)
}

func (i *Invoker[Tx, I, O]) getContext(c *echo.Context) context.Context {
	return i.ctx(c)
}

func (i *Invoker[Tx, I, O]) getInputMapper() func(c *echo.Context) (*I, error) {
	if i.inputMapper == nil {
		return Zero[I]
	}
	return i.inputMapper
}

func (i *Invoker[Tx, I, O]) getOutputMapper() func(*echo.Context, *O) error {
	if i.outputMapper == nil {
		return WriteJSON[O]
	}
	return i.outputMapper
}
