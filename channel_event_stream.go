package render

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gdey/chi-render/responders/helpers"
)

func ChannelEventStream(w http.ResponseWriter, r *http.Request, v interface{}) error {

	if reflect.TypeOf(v).Kind() != reflect.Chan {
		panic(fmt.Sprintf("render: event stream expects a channel, not %v", reflect.TypeOf(v).Kind()))
	}

	helpers.SetContentTypeHeader(w, "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	if r.ProtoMajor == 1 {
		// An endpoint MUST NOT generate an HTTP/2 message containing connection-specific header fields.
		// Source: RFC7540
		w.Header().Set("Connection", "keep-alive")
	}

	w.WriteHeader(http.StatusOK)

	ctx := r.Context()
	for {
		switch chosen, recv, ok := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(v)},
		}); chosen {
		case 0: // equivalent to: case <-ctx.Done()
			w.Write([]byte("event: error\ndata: {\"error\":\"Server Timeout\"}\n\n"))
			w.WriteHeader(http.StatusGatewayTimeout)
			return nil

		default: // equivalent to: case v, ok := <-stream
			if !ok {
				w.Write([]byte("event: EOF\n\n"))
				return nil
			}
			v := recv.Interface()

			// Build each channel item.
			if rv, ok := v.(Renderer); ok {
				err := renderer(w, r, rv)
				if err != nil {
					v = err
				} else {
					v = rv
				}
			}

			bytes, err := json.Marshal(v)
			if err != nil {
				w.Write([]byte(fmt.Sprintf("event: error\ndata: {\"error\":\"%v\"}\n\n", err)))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				continue
			}
			w.Write([]byte(fmt.Sprintf("event: data\ndata: %s\n\n", bytes)))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
