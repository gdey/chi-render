package responders

import "net/http"

// M is a convenience alias for quickly building a map structure that is going
// out to a responder. Just a short-hand.
type M map[string]interface{}

// Func defined a function that will take an object and Marshal it into a content type
// before writing it to the http.ResponseWriter
type Func func(http.ResponseWriter, *http.Request, interface{}) error
