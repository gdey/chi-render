package helpers

import (
	"context"
	"net/http"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "chi render context value " + k.name
}

var (
	// StatusCtxKey is a context key to record a future HTTP response status code.
	StatusCtxKey = &contextKey{"Status"}
	// ContentTypeCtxKey is a context for recording a by-pass for context-type
	ContentTypeCtxKey = &contextKey{"ContentType"}
	// RenderCtxKey is a context for getting the render to use
	RenderCtxKey = &contextKey{name: "Renderer"}
)

// Status sets a HTTP response status code hint into request context at any point
// during the request life-cycle. Before the Responder sends its response header
// it will check the StatusCtxKey
func Status(r *http.Request, status int) {
	*r = *r.WithContext(context.WithValue(r.Context(), StatusCtxKey, status))
}

type headerer interface {
	Header() http.Header
}

type writeHeaderer interface {
	WriteHeader(int)
}

func SetNoSniffHeader(w headerer) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func SetContentTypeHeader(w headerer, value string) {
	w.Header().Set("Content-Type", value)
}

func WriteStatus(w writeHeaderer, ctx context.Context) {
	if status, ok := ctx.Value(StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
}

// NoContent returns a HTTP 204 "No Content" response.
func NoContent(w writeHeaderer) {
	w.WriteHeader(204)
}
