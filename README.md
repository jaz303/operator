# operator

`operator` is a Go coordination library for invoking transactional operations, intended for use in DDD-like systems.

Additionally, an `httpbind` package is provided for easily exposing Operations to stdlib-compliant HTTP handlers. Sub-packages for non-standard routers such as Echo will be added in the future.



## `httpbind`

```golang
// Hub - instantiated elsewhere
var hub *operator.Hub[MyTx]

// The operation implementation
func MyOp(ctx *operator.OpContext[MyTx], input *MyInput) (*MyOutput, error) {
    tx, err := ctx.Tx()
    if err != nil {
        // failed to start transaction
        return nil, err
    }

    // ...
    // do work inside transaction
    // ...

    return &MyOutput{
        Foo: input.Foo,
    }, nil
}

func MyHandler(w http.ResponseWriter, r *http.Request) {
    var params = struct{
        Foo string `json:"foo"`
    }{}

    // This example is illustrative but possibly over-engineered for simple mappings.
    // If your operation input and outputs map trivially to JSON you can just use the
    // simpler httpbind.ParseJSON and httpbind.WriteJSON functions.
    httpbind.Bind(hub, MyOp).
        WithInputMapper(httpbind.IndirectJSONInput[params, MyInput](func(ps *params) {
            return &MyInput{
                Foo: params.Foo,
            }, nil
        })).
        WithOutputMapper(func(w http.ResponseWriter, o *MyOutput) {
            json.NewEncoder(w).Encode(map[string]any{
                "foo": o.Foo,
            })
        }).
        Go(w, r)
}
```

