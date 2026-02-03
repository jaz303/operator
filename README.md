# operator

`operator` is a small, opinionated Go library for running application operations with clear
transactional boundaries, synchronous domain events, and simple adapters for exposing
operations via HTTP.

`operator` is a disciplined way to execute business logic:

  - every request runs inside an operation
  - transactions are lazy and explicit
  - domain events are synchronous and transactional
  - side effects run only after commit
  - adapters (HTTP, CLI, Lambda etc) stay thin

Additionally, an `httpbind` package is provided for easily exposing Operations to stdlib-compliant HTTP handlers. Sub-packages for non-standard routers such as Echo will be added in the future.

## Core Concepts

### Operation

```golang
type Operation[Tx, I, O] func(ctx *operator.OpContext[Tx], input *I) (*O, error)
```

An operation is a unit of application work, with typed input/output, explicit access to an operation
context, and no hidden global state. If an operation returns an error (or panics), everything rolls
back.

### Transactions

Transactions do not start automatically, they instead beging lazily on first use:

```golang
tx, err := ctx.Tx()
```

Transactions are implemented by adopting a simple 2-method interface: `Commit(context.Context)`
and `Rollback(context.Context)` so it's

### Hub

```golang
hub := operator.NewHub(beginTransaction)
```

`Hub` is the `operator`'s central configuration object. It knows how to start a transaction,


### Domain Events

```golang
ctx.Emit(UserCreated{ID: id})
```

Events are buffered until the operation completes, then dispatched synchronously before commit.

Domain events are part of your consistency boundary: if an event handler fails, the entire
transaction is rolled back.

### After-Commit Hooks

```golang
ctx.AfterFunc(func(ctx *OpContext[Tx]) {
    sendWelcomeEmail(...)
})
```

After-commit hooks run only after an operation succeeds (and after commit if, a transaction was
started). They are intended for side-effects such as sending emails, enqueuing jobs, or triggering
webhooks.

## Basic Usage Example

### 1. Define a transaction type

First thing we need is a transaction type, since everything else in `operator` is generic with
respect to it. Assuming we're using `database/sql`, we can easily adapt an `*sql.Tx` to match
`operator`'s requirements:

```golang
type Tx struct {
    *sql.Tx
}

func (t Tx) Commit(ctx context.Context) error { return t.Tx.Commit() }
func (t Tx) Rollback(ctx context.Context) error { return t.Tx.Rollback() }
```

The adapter is necessary because `operator`'s methods accept a `context.Context`
(__Note:__ if you're using `pgx`, its transaction type will drop right in without the need
for an adapter!)

### 2. Create a Hub

Next we need a `Hub`. This fulfils two roles:

  1. provides a place from where transactions can be created
  2. maintains a registry of event handlers

__Note:__ event handlers are permitted to access the operation's transaction.
This is fine - __event handling is part of the consistency boundary__.

```golang
hub := operator.NewHub(func(ctx context.Context) (Tx, error) {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil { return Tx{}, err }
    return Tx{tx}, nil
})

hub.RegisterEventHandler(&UserCreated{}, func(ctx *operator.OpContext[Tx], evt *UserCreated) error {
    tx, err := ctx.Tx()
    if err != nil { return err }
    return insertAuditLog(tx, "user created", evt.ID)
})
```

### 3. Define an Operation

An Operation is just a Go function that accepts an `*operator.OpContext[Tx]` and input arguments,
and returns an output value/error.

Note that operation implementations never commit or roll back transactions directly -
`operator` takes care of this as part of its operation invocation handler.

```golang
type CreateUserInput struct {
    Email string
}

type CreateUserOutput struct {
    ID int64
}

func CreateUser(ctx *operator.OpContext[Tx], in *CreateUserInput) (*CreateUserOutput, error) {
    tx, err := ctx.Tx()
    if err != nil { return nil, err }

    id, err := insertUser(tx, in.Email)
    if err != nil { return nil, err }

    ctx.Emit(&UserCreated{ID: id})

    return &CreateUserOutput{ID: id}, nil
}
```

### 4. Invoke the Operation

```golang
out, err := operator.Invoke(ctx, hub, CreateUser, &CreateUserInput{
    Email: "test@example.com",
})
```

That's all there is to it - transaction handling and event dispatch is handled automatically.

## Binding Operations to HTTP

The `httpbind` package exposes operations over HTTP without leaking concerns - zero business logic in the handlers.

```golang
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    httpbind.Bind(hub, CreateUser).
        WithInputMapper(httpbind.ParseJSON[CreateUserInput]).
        WithJSONOutput(func(w http.ResponseWriter, out *CreateUserOutput) any {
            return map[string]any{"id": out.ID}
        }).
        Go(w, r)
}
```

Input, output, and error mapping is fully configurable and can be as simple or as complex as you need. Whether your input
and output types map directly to JSON, or if you require something deeper, `operator` can adapt.

## Copyright & License

&copy: 2026 Jason Frame, licensed under the MIT license.
