package responders

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/gdey/chi-render/responders/helpers"
)

// PlainText writes a string to the response, setting the Content-Type as
// text/plain.
func PlainText(w http.ResponseWriter, r *http.Request, v interface{}) error {
	var txt string

	switch vv := v.(type) {
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
	helpers.SetContentTypeHeader(w, "text/plain; charset=utf-8")
	helpers.WriteStatus(w, r.Context())

	w.Write([]byte(txt))

	return nil
}

// Data writes raw bytes to the response, setting the Content-Type as
// application/octet-stream.
func Data(w http.ResponseWriter, r *http.Request, v interface{}) {

	helpers.SetNoSniffHeader(w)
	helpers.SetContentTypeHeader(w, "application/octet-stream")
	helpers.WriteStatus(w, r.Context())

	var (
		b   []byte
		err error
	)

	switch vv := v.(type) {
	case encoding.BinaryMarshaler:
		b, err = vv.MarshalBinary()
		if err != nil {
			return
		}
	case []byte:
		b = vv
	case encoding.TextMarshaler:
		t, err := vv.MarshalText()
		if err != nil {
			return
		}
		b = t
	case string:
		b = []byte(vv)
	case fmt.Stringer:
		b = []byte(vv.String())

	default:
		binary.Write(w, binary.BigEndian, v)
		return
	}
	w.Write(b)
	return
}
