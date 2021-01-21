package decoders

import "io"

// Func describes a function used to decode a byte slice into the given
// object
type Func func(r io.Reader, v interface{}) error
