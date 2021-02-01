// The render package helps manage HTTP request and
// response payloads.
//
// Every well-designed, robust and maintainable Web Service /
// REST API also needs well-defined request and payloads.
// Together with the endpoint handler, the request and response
// payloads make up the contract between your server and the
// clients calling on it.
//
// Typically in a REST API application, you will have data
// models (objects/structs) that hold lower-level runtime
// application state, and at time you need to assemble, decorate
// hide or transform the representation before responding to
// a client. That server output (response payload) structure,
// is likely the input structure to another handler on the server.
//
// This is where render comes in - offering a few simple helpers and
// to provide a simple pattern for managing payload encoding and
// decoding.
//
//  The typical flow will look like the following:
//
//    +-------------+
//    | REQUEST     | // The in coming request from the client
//    +-----+-------+
//          |
//          V
//    +-------------+ // Determined by the Content type of the body
//    | decoders    | // in order to decode the body into the passed
//    +-----+-------+ // into the provided object/struct (decoders package)
//          |
//          V
//    +-------------+ // Modify, assemble, decorate based models off the
//    | Binder      | // decoded object/struct from the decoder
//    +-----+-------+
//          |
//          V
//    +-------------+
//    | Application |
//    +-----+-------+
//          |
//          V
//    +-------------+ // Modify, assemble, decorate  models to prepare
//    | Render      | // them to be encoded by the Responder
//    +-----+-------+
//          |
//          V
//    +-------------+ // Determined by the Content-Type of the response
//    | responders  | // object, encode the provided object/struct (responders package)
//    +-----+-------+
//          |
//          V
//    +-------------+
//    | RESPONSE    |
//    +-------------+
//
package render

import (
	"context"
	"net/http"
	"reflect"

	"github.com/gdey/chi-render/responders/helpers"

	"github.com/gdey/chi-render/decoders"
	"github.com/gdey/chi-render/responders"
)

// Renderer interface for managing response payloads.
type Renderer interface {
	// Render should modify the object so that it is in the correct configuration
	// for the responders to render the object. One can interrogate the request object
	// or if necessary modify the headers in the ResponseWriter object.
	// The Render method should not write to the body fo the ResponseWriter object,
	// the at is reserved for the responder objects.
	Render(w http.ResponseWriter, r *http.Request) error
}

// Binder interface for managing request payloads.
type Binder interface {
	// Binder should be used to recompose the original the data model object.
	// The Binder function is called after the decoders is called so the body
	// of the http.Request object will be spent.
	Bind(r *http.Request) error
}

// FromContext will retrieve the render object from the context
func FromContext(r *http.Request) *Controller {

	ctx := r.Context()
	if ctx == nil {
		return nil
	}

	ren, _ := ctx.Value(helpers.RenderCtxKey).(*Controller)
	return ren
}

// WithCtx is the middleware to attach a new render.Controller to the context
func WithCtx(c *Controller) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*r = *r.WithContext(context.WithValue(r.Context(), helpers.RenderCtxKey, c))
			next.ServeHTTP(w, r)
		})
	}
}

// CloneDefault will return a Clone of the default controller
func CloneDefault() *Controller { return defaultCtrl.Clone() }

// NilRender is an empty struct that can be embedded to provide a simple
// way to turn a struct into a Render-able object
type NilRender struct{}

// Render does nothing
func (NilRender) Render(_ http.ResponseWriter, _ *http.Request) error { return nil }

// NilBinder is an empty struct that can be embedded to provide a simple
// way to return a struct into a Bind-able object
type NilBinder struct{}

// Bind does nothing
func (NilBinder) Bind(_ *http.Request) error { return nil }

// Bind decodes a request body and executes the Binder method of the
// payload structure.
func Bind(r *http.Request, v Binder) error { return defaultCtrl.Bind(r, v) }

// Render renders a single payload and respond to the client request.
func Render(w http.ResponseWriter, r *http.Request, v Renderer) error {
	return defaultCtrl.Render(w, r, v)
}

// RenderList renders a slice of payloads and responds to the client request.
func RenderList(w http.ResponseWriter, r *http.Request, l []Renderer) error {
	return defaultCtrl.RenderList(w, r, l)
}

// SetDecoder will set the decoder for the given content type.
// Use a nil DecodeFunc to unset a content type
func SetDecoder(contentType ContentType, decoder decoders.Func) {
	_ = defaultCtrl.SetDecoder(contentType, decoder)
}

// SupportedDecoders returns a ContentTypeSet of the configured Content types with decoders
func SupportedDecoders() *ContentTypeSet { return defaultCtrl.SupportedDecoders() }

// SetResponder will set the responder for the given content type.
// Use a nil RespondFunc to unset a content type
func SetResponder(contentType ContentType, responder responders.Func) {
	_ = defaultCtrl.SetResponder(contentType, responder)
}

// SupportedResponders returns a ContentTypeSet of the configured Content types with responders
func SupportedResponders() *ContentTypeSet { return defaultCtrl.SupportedResponders() }

// Status sets a HTTP response status code hint into request context at any point
// during the request life-cycle. Before the Responder sends its response header
// it will check the StatusCtxKey
func Status(r *http.Request, status int) { helpers.Status(r, status) }

func isNil(f reflect.Value) bool {
	switch f.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return f.IsNil()
	default:
		return false
	}
}

// Executed top-down
func renderer(w http.ResponseWriter, r *http.Request, v Renderer) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// We call it top-down.
	if err := v.Render(w, r); err != nil {
		return err
	}

	// We're done if the Renderer isn't a struct object
	if rv.Kind() != reflect.Struct {
		return nil
	}

	// For structs, we call Render on each field that implements Renderer
	for i := 0; i < rv.NumField(); i++ {

		f := rv.Field(i)
		// if the field is nil weather it's a Render or not we should skip
		if isNil(f) {
			continue
		}

		// Check to see if it's a render type
		if f.Type().Implements(rendererType) {
			fv := f.Interface().(Renderer)
			if err := renderer(w, r, fv); err != nil {
				return err
			}
			continue
		}

		// Are we dealing with a slice or an array of objects
		// that are renders?
		if f.Kind() != reflect.Slice && f.Kind() != reflect.Array {
			// No we are not continue
			continue
		}

		length := f.Len()
		if length == 0 {
			continue
		}

		// We know we have at least one
		rvv := f.Index(0)
		// Let's check to see if we have an interface we are dealing with.
		// or a set of known values.
		isInterface := rvv.Kind() == reflect.Interface
		if rvv.Type().Implements(rendererType) {
			fv := rvv.Interface().(Renderer)
			if err := renderer(w, r, fv); err != nil {
				return err
			}
		} else if !isInterface {
			// No need to scan through the rest of the array
			continue
		}

		for j := 1; j < length; j++ {
			rvv = f.Index(j)
			if isInterface && !rvv.Type().Implements(rendererType) {
				// skip this one
				continue
			}
			fv := rvv.Interface().(Renderer)
			if err := renderer(w, r, fv); err != nil {
				return err
			}
		}

	}

	return nil
}

// Executed bottom-up
func binder(r *http.Request, v Binder) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// Call Binder on non-struct types right away
	if rv.Kind() != reflect.Struct {
		return v.Bind(r)
	}

	// For structs, we call Bind on each field that implements Binder
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)

		if isNil(f) {
			continue
		}

		if f.Type().Implements(binderType) {
			fv := f.Interface().(Binder)
			if err := binder(r, fv); err != nil {
				return err
			}

			continue
		}

		// Maybe we are dealing with a slice or an Array?
		if f.Kind() != reflect.Slice && f.Kind() != reflect.Array {
			// No we are not continue
			continue
		}

		length := f.Len()
		if length == 0 {
			continue
		}

		// We know we have at least one
		rvv := f.Index(0)
		// Let's check to see if we have an interface we are dealing with.
		// or a set of known values.
		isInterface := rvv.Kind() == reflect.Interface
		if rvv.Type().Implements(binderType) {
			fv := rvv.Interface().(Binder)
			if err := binder(r, fv); err != nil {
				return err
			}
		} else if !isInterface {
			// No need to scan through the rest of the array
			continue
		}

		for j := 1; j < length; j++ {
			rvv = f.Index(j)
			if isInterface && !rvv.Type().Implements(binderType) {
				// skip this one
				continue
			}
			fv := rvv.Interface().(Binder)
			if err := binder(r, fv); err != nil {
				return err
			}
		}

	}

	// We call it bottom-up
	if err := v.Bind(r); err != nil {
		return err
	}

	return nil
}

var (
	rendererType = reflect.TypeOf(new(Renderer)).Elem()
	binderType   = reflect.TypeOf(new(Binder)).Elem()

	// Make sure controller fulfill the Interface interface
	_ = Interface(new(Controller))
	_ = Renderer(NilRender{})
	_ = Binder(NilBinder{})
	_ = Renderer(struct{ NilRender }{})
	_ = Binder(struct{ NilBinder }{})
)
