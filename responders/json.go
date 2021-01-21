package responders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gdey/chi-render/responders/helpers"
	"net/http"
)


// JSON marshals 'v' to JSON, automatically escaping HTML and setting the
// Content-Type as application/json.
func JSON(w http.ResponseWriter, r *http.Request, v interface{}) error {

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("JSON encode: %w", err)
	}

	helpers.SetNoSniffHeader(w)
	helpers.SetContentTypeHeader(w,"application/json; charset=utf-8")
	helpers.WriteStatus(w,r.Context())
	_, _ = w.Write(buf.Bytes())

	return nil
}
