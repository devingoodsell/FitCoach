// Package httpx provides the HTTP router, middleware, and JSON/error helpers
// shared by every backend handler. Domain packages register routes here; they
// do not talk to net/http directly beyond the http.Handler signature.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Router wraps http.ServeMux with a middleware chain applied to every request.
// Go 1.22+ method+pattern routing means we need no third-party router.
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// Middleware wraps an http.Handler, returning a new one. Middlewares are applied
// outermost-first in the order they were added via Use.
type Middleware func(http.Handler) http.Handler

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// Use appends a middleware to the chain. Call before Handle/HandleFunc.
func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// Handle registers a handler for a method+path pattern, e.g. "POST /auth/login".
func (r *Router) Handle(pattern string, h http.Handler) {
	r.mux.Handle(pattern, h)
}

// HandleFunc registers a handler function for a method+path pattern.
func (r *Router) HandleFunc(pattern string, h http.HandlerFunc) {
	r.mux.Handle(pattern, h)
}

// ServeHTTP applies the middleware chain and dispatches to the mux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var h http.Handler = r.mux
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	h.ServeHTTP(w, req)
}

// WriteJSON encodes v as JSON with the given status code. A nil body writes the
// status with no payload.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// ErrorBody is the stable error envelope returned to clients. Messages are
// deliberately generic where enumeration matters (auth); callers control text.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// WriteError writes a JSON error envelope with the given status. code is a short
// machine-readable token (e.g. "invalid_request"); msg is human-readable.
func WriteError(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, ErrorBody{Error: code, Message: msg})
}

// DecodeJSON reads and strictly decodes the request body into dst. Unknown
// fields are rejected so contract drift surfaces early.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
