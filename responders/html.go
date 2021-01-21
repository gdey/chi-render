package responders

import (
	"encoding"
	"fmt"
	"net/http"

	"github.com/gdey/chi-render/responders/helpers"
)

type HTMLMarshaler interface {
	MarshalHTML() ([]byte, error)
}

// HTML writes a string to the response, setting the Content-Type as text/html.
func HTML(w http.ResponseWriter, r *http.Request, v interface{}) error {
	var txt string

	switch vv := v.(type) {
	case HTMLMarshaler:
		btxt, err := vv.MarshalHTML()
		if err != nil {
			return err
		}
		txt = string(btxt)

	case encoding.TextMarshaler:
		btxt, err := vv.MarshalText()
		if err != nil {
			return err
		}
		txt = string(btxt)
	case string:
		txt = vv
	case fmt.Stringer:
		txt = vv.String()
	default:
		return ErrCanNotEncodeObject
	}

	helpers.SetNoSniffHeader(w)
	helpers.SetContentTypeHeader(w, "text/html; charset=utf-8")
	helpers.WriteStatus(w, r.Context())
	w.Write([]byte(txt))

	return nil
}
