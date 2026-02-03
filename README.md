# operator

`operator` is a small, opinionated Go library for running application operations with clear
transactional boundaries, synchronous domain events, and simple adapters for exposing
operations via HTTP.

`operator` is a disciplined way to execute business logic:

  - every request runs inside an operation, decoupled from HTTP
  - transactions are lazy and explicit
  - domain events are synchronous and transactional
  - side effects run only after commit
  - adapters (HTTP, CLI, Lambda etc) stay thin

## What is a "Transaction"?

In `operator`, a *transaction* is an in-process consistency boundary - typically an SQL transaction.

This library is designed for systems where:

  - application state lives primarily in one database
  - transactional guarantees are provided by that database
  - all work happens within a single process

Distributed transactions, two-phase commit, sagas, and cross-system coordination are
explicitly out of scope.

That said, `operator` is compatible with common patterns for crossing process boundaries.
In particular, synchronous domain events make it easy to implement patterns such as the
[transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html),
where events are recorded transactionally and picked up by external systems using techniques
such as CDC.

The goal is not to solve distributed consistency, but to make local consistency boring.

## When Should You Use `operator`?

`operator` is a good fit when:

  - your service has non-trivial business logic
  - you care about transactional correctness
  - you want domain events to be part of your consistency model
  - you don't want transactional mechanisms leaking into HTTP handlers
  - you prefer explicit control over framework magic

It works especially well for CRUD-plus systems that have grown beyond "simple handlers",
where concerns like auditing, invariants, and side effects start to accumulate.

You probably *don’t* need `operator` if:

  - your service is a thin proxy or simple read-only API
  - you rely heavily on distributed workflows or eventual consistency
  - your persistence layer doesn’t support transactions at all

`operator` doesn’t try to be a framework. It provides a small, well-defined execution model
for application services, and then gets out of the way.

## Core Concepts

### Operation

```golang
type Operation[Tx, I, O] func(ctx *operator.OpContext[Tx], input *I) (*O, error)
```

An operation is a unit of application work, with typed input/output, explicit access to an operation
context, and no hidden global state. If an operation returns an error (or panics), everything rolls
back.

### Transactions

Not every operation requires a persitence, so transactions do not start automatically.
Instead they are initialised lazily on first use:

```golang
tx, err := ctx.Tx()
```

Transactions are implemented by adopting a simple 2-method interface: `Commit(context.Context)`
and `Rollback(context.Context)` so it's trivial to adapt `operator` to whatever persistence
system you're using.

### Hub

```golang
hub := operator.NewHub(beginTransaction)
```

`Hub` is the `operator`'s central configuration object. It knows how to start a transaction, as well as
maintaining a registry of event handlers. All operations are invoked through a `Hub`.

### Domain Events

```golang
ctx.Emit(&UserCreated{ID: id})
```

Domain events enable an application's subdomains to react to actions that occur in other parts of the system.

Events are buffered until the emitting operation completes, and then dispatched to all registered handlers
synchronously, before commit. If any event handler fails, the entire transaction is rolled back. Thus,
events exist with `operator`'s consistency boundary - they are not simply "fire and forget".

### After-Commit Hooks

```golang
ctx.AfterFunc(func(ctx *OpContext[Tx]) {
    sendWelcomeEmail(...)
})
```

After-commit hooks run only after an operation succeeds (and after commit, if a transaction was
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
and returns an output value or error.

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

At the moment, only the stdlib's HTTP handler signature is supported - support for more frameworks will be added soon (PRs
gladly accepted!).

## Copyright & License

&copy; 2026 Jason Frame, licensed under the MIT license.
