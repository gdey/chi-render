package decoders

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

func JSON(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	return json.NewDecoder(r).Decode(v)
}
