package responders

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/gdey/chi-render/responders/helpers"
	"net/http"
)

// XML marshals 'v' to XML, setting the Content-Type as application/xml. It
// will automatically prepend a generic XML header (see encoding/xml.Header) if
// one is not found in the first 100 bytes of 'v'.
func XML(w http.ResponseWriter, r *http.Request, v interface{}) error {
	b, err := xml.Marshal(v)
	if err != nil {
		return fmt.Errorf("XML marshal: %w", err)
	}

	helpers.SetNoSniffHeader(w)
	helpers.SetContentTypeHeader(w,"application/xml; charset=utf-8")
	helpers.WriteStatus(w,r.Context())

	// Try to find <?xml header in first 100 bytes (just in case there are some XML comments).
	findHeaderUntil := len(b)
	if findHeaderUntil > 100 {
		findHeaderUntil = 100
	}

	if !bytes.Contains(b[:findHeaderUntil], []byte("<?xml")) {
		// No header found. Print it out first.
		w.Write([]byte(xml.Header))
	}

	w.Write(b)
	return nil
}

