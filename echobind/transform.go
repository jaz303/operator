package echobind

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// BindJSON parses the request body into a *P using Echo's Bind
func BindJSON[P any](c *echo.Context) (*P, error) {
	var out P
	if err := c.Bind(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IndirectJSONInput parses the request body into a *P before passing it to a
// user-defined transformer function that produces a *I.
func IndirectJSONInput[P any, I any](t func(*P) (*I, error)) func(c *echo.Context) (*I, error) {
	return func(c *echo.Context) (*I, error) {
		params, err := BindJSON[P](c)
		if err != nil {
			return nil, err
		}
		return t(params)
	}
}

// WriteJSON writes a *T to the response as JSON with status 200
func WriteJSON[T any](c *echo.Context, val *T) error {
	return c.JSON(http.StatusOK, val)
}

// Transform returns a function that reads input from an Echo context as an *I
// before passing it to a user-defined transformer function that produces an *O.
func Transform[I any, O any](getInput func(c *echo.Context) (*I, error), t func(*I) (*O, error)) func(c *echo.Context) (*O, error) {
	return func(c *echo.Context) (*O, error) {
		input, err := getInput(c)
		if err != nil {
			return nil, err
		}
		return t(input)
	}
}

// Identity is a passthrough transformer
func Identity[I any](in *I) (*I, error) {
	return in, nil
}

// Zero returns a zero-value input
func Zero[I any](c *echo.Context) (*I, error) {
	var out I
	return &out, nil
}
