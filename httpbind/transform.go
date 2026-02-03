package httpbind

import (
	"encoding/json"
	"net/http"
)

// ParseJSON parses r's Body into a *P
func ParseJSON[P any](r *http.Request) (*P, error) {
	var out P
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IndirectJSONINput parses r's Body into a *P before passing it to a
// user-defined transformer function that produces a *I.
func IndirectJSONInput[P any, I any](t func(*P) (*I, error)) func(r *http.Request) (*I, error) {
	return func(r *http.Request) (*I, error) {
		params, err := ParseJSON[P](r)
		if err != nil {
			return nil, err
		}
		return t(params)
	}
}

// WriteJSON writes a *T to w as JSON
func WriteJSON[T any](w http.ResponseWriter, val *T) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}

// Transform returns a function that reads input from an HTTP request as an *I
// before passing it to a user-defined transformer function that produces an *O.
func Transform[I any, O any](getInput func(r *http.Request) (*I, error), t func(*I) (*O, error)) func(r *http.Request) (*O, error) {
	return func(r *http.Request) (*O, error) {
		input, err := getInput(r)
		if err != nil {
			return nil, err
		}
		return t(input)
	}
}

func Identity[I any](in *I) (*I, error) {
	return in, nil
}

func Zero[I any](r *http.Request) (*I, error) {
	var out I
	return &out, nil
}
