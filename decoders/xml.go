package decoders

import (
	"encoding/xml"
	"io"
	"io/ioutil"
)

func XML(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	return xml.NewDecoder(r).Decode(v)
}
