package render

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"sync"

	"github.com/gdey/chi-render/responders/helpers"

	"github.com/gdey/chi-render/decoders"

	"github.com/gdey/chi-render/responders"
)

var (
	// ErrControllerIsNil will be returned by methods that require the Controller object
	// to be not nil
	ErrControllerIsNil = errors.New("controller is nil")

	// defaultCtrl is the default controller that is used if a controller is nil,
	// or the package functions are used.
	defaultCtrl = Controller{
		responders: map[ContentType]responders.Func{
			ContentTypeDefault:     responders.JSON,
			ContentTypeJSON:        responders.JSON,
			ContentTypeXML:         responders.XML,
			ContentTypeEventStream: ChannelEventStream,
		},
		decoders: map[ContentType]decoders.Func{
			ContentTypeJSON: decoders.JSON,
			ContentTypeXML:  decoders.XML,
		},
		DefaultRequest:  ContentTypeNone,
		DefaultResponse: ContentTypeDefault,
	}
)

// Interface defines what a render controller should behave like
type Interface interface {
	Bind(r *http.Request, v Binder) error
	Render(w http.ResponseWriter, r *http.Request, v Renderer) error
	RenderList(w http.ResponseWriter, r *http.Request, l []Renderer) error
}

// Controller is responsible for managing the respond types that are available
type Controller struct {
	responderLck sync.RWMutex
	// responders is a mapping of content type to a function that can
	//  marshal an object to that content type
	responders map[ContentType]responders.Func

	decoderLck sync.RWMutex
	// decoders is a mapping content type to a function that can
	// unmarshal a byte slice to an object
	decoders map[ContentType]decoders.Func

	// If no content type matches, this content type will be used.
	DefaultRequest ContentType
	// If no Accept header match, this content type will be used to render the object
	DefaultResponse ContentType
}

// Status sets a HTTP response status code hint into request context at any point
// during the request life-cycle. Before the Responder sends its response header
// it will check the StatusCtxKey
func (ctrl *Controller) Status(r *http.Request, status int) { helpers.Status(r, status) }

// Clone will return a deep copy version of the controller
// if ctrl is nil a clone of the default system controller will
// be returned instead
func (ctrl *Controller) Clone() *Controller {
	if ctrl == nil {
		return defaultCtrl.Clone()
	}
	child := new(Controller)
	child.DefaultResponse = ctrl.DefaultResponse
	child.DefaultRequest = ctrl.DefaultRequest
	child.responders = make(map[ContentType]responders.Func, len(ctrl.responders))
	child.decoders = make(map[ContentType]decoders.Func, len(ctrl.decoders))
	ctrl.responderLck.RLock()
	for name, val := range ctrl.responders {
		child.responders[name] = val
	}
	ctrl.responderLck.RUnlock()
	ctrl.decoderLck.RLock()
	for name, val := range ctrl.decoders {
		child.decoders[name] = val
	}
	ctrl.decoderLck.RUnlock()
	return child
}

// Render renders a single payload and respond to the client request.
func (ctrl *Controller) Render(w http.ResponseWriter, r *http.Request, v Renderer) error {
	if ctrl == nil {
		return defaultCtrl.Render(w, r, v)
	}
	if err := renderer(w, r, v); err != nil {
		return err
	}
	ctrl.respond(w, r, v)
	return nil
}

// RenderList renders a slice of payloads and responds to the client request.
func (ctrl *Controller) RenderList(w http.ResponseWriter, r *http.Request, l []Renderer) error {
	if ctrl == nil {
		return defaultCtrl.RenderList(w, r, l)
	}
	for _, v := range l {
		if err := renderer(w, r, v); err != nil {
			return err
		}
	}
	ctrl.respond(w, r, l)
	return nil
}

// channelIntoSlice buffers channel data into a slice.
func channelIntoSlice(w http.ResponseWriter, r *http.Request, from interface{}) interface{} {
	ctx := r.Context()

	var to []interface{}
	for {
		switch chosen, recv, ok := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(from)},
		}); chosen {
		case 0: // equivalent to: case <-ctx.Done()
			http.Error(w, "Server Timeout", 504)
			return nil

		default: // equivalent to: case v, ok := <-stream
			if !ok {
				return to
			}
			v := recv.Interface()

			// Render each channel item.
			if rv, ok := v.(Renderer); ok {
				err := renderer(w, r, rv)
				if err != nil {
					v = err
				} else {
					v = rv
				}
			}

			to = append(to, v)
		}
	}
}

func (ctrl *Controller) respond(w http.ResponseWriter, r *http.Request, v interface{}) {
	var err error

	acceptedTypes := GetAcceptedContentType(r)
	if v != nil {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Chan:
			if acceptedTypes.Has(ContentTypeEventStream) {
				ctrl.responderLck.RLock()
				fn, ok := ctrl.responders[ContentTypeEventStream]
				ctrl.responderLck.RUnlock()
				if ok {
					if err = fn(w, r, v); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					return
				}
			}
			v = channelIntoSlice(w, r, v)
		}
	}

	for acceptedTypes.Next() {
		// Skip ContentTypeEventStream, handled up top.
		if acceptedTypes.Type() == ContentTypeEventStream {
			continue
		}
		ct := acceptedTypes.Type()
		ctrl.responderLck.RLock()
		fn, ok := ctrl.responders[ct]
		ctrl.responderLck.RUnlock()
		if !ok {
			continue
		}

		if err = fn(w, r, v); err != nil {

			if errors.Is(err, responders.ErrCanNotEncodeObject) {
				// Let's try the next content type
				continue
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	ctrl.responderLck.RLock()
	if ctrl.DefaultResponse == "" {
		ctrl.DefaultResponse = ContentTypeDefault
	}
	fn, ok := ctrl.responders[ctrl.DefaultResponse]
	ctrl.responderLck.RUnlock()

	if !ok {
		panic("Default Controller Responder not set!")
	}
	if err = fn(w, r, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SetResponder will set the responder for the given content type.
// Use a nil RespondFunc to unset a content type
// Only error this function will return is ErrControllerIsNil; is returned
// if the Controller object is nil.
func (ctrl *Controller) SetResponder(contentType ContentType, responder responders.Func) error {
	if ctrl == nil {
		return ErrControllerIsNil
	}
	ctrl.responderLck.Lock()
	ctrl.responders[contentType] = responder
	ctrl.responderLck.Unlock()
	return nil
}

// SupportedResponders returns a ContentTypeSet of the configured Content types with responders
func (ctrl *Controller) SupportedResponders() *ContentTypeSet {
	if ctrl == nil {
		return defaultCtrl.SupportedResponders()
	}

	ctrl.responderLck.RLock()
	stringValues := make([]string, 0, len(ctrl.responders))
	for value := range ctrl.responders {
		stringValues = append(stringValues, string(value))
	}
	ctrl.responderLck.RUnlock()

	sort.Strings(stringValues)
	return NewContentTypeSet(stringValues...)
}

// Bind decodes a request body and executes the Binder method of the
// payload structure.
func (ctrl *Controller) Bind(r *http.Request, v Binder) error {
	if ctrl == nil {
		return defaultCtrl.Bind(r, v)
	}
	if err := ctrl.decode(r, v); err != nil {
		return err
	}
	return binder(r, v)
}

func (ctrl *Controller) decode(r *http.Request, v interface{}) error {

	ct := GetRequestContentType(r, ctrl.DefaultRequest)

	ctrl.decoderLck.RLock()
	decoder := ctrl.decoders[ct]
	ctrl.decoderLck.RUnlock()

	if decoder != nil {
		return decoder(r.Body, v)
	}
	return fmt.Errorf("render: unable to automatically decode the request content type: '%s'", ct)
}

// SetDecoder will set the decoder for the given content type.
// Use a nil DecodeFunc to unset a content type
// Only error this function will return is ErrControllerIsNil; is returned
// if the Controller object is nil.
func (ctrl *Controller) SetDecoder(contentType ContentType, decoder decoders.Func) error {
	if ctrl == nil {
		return ErrControllerIsNil
	}
	ctrl.decoderLck.Lock()
	ctrl.decoders[contentType] = decoder
	ctrl.decoderLck.Unlock()
	return nil
}

// SupportedDecoders returns a ContentTypeSet of the configured Content types with decoders
func (ctrl *Controller) SupportedDecoders() *ContentTypeSet {
	if ctrl == nil {
		return defaultCtrl.SupportedDecoders()
	}

	ctrl.decoderLck.RLock()
	stringValues := make([]string, 0, len(ctrl.decoders))
	for value := range ctrl.decoders {
		stringValues = append(stringValues, string(value))
	}
	ctrl.decoderLck.RUnlock()
	sort.Strings(stringValues)
	return NewContentTypeSet(stringValues...)
}
