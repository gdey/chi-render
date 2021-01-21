package responders

import "errors"

var (
	// ErrCanNotEncodeObject should be returned by RespondFunc if the Responder should
	// try a different content type, as we don't know how to respond with this object
	ErrCanNotEncodeObject = errors.New("error can not encode object")
)
