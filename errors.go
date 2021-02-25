package render

import (
	"crypto/rand"
	"log"
	"net/http"
)

const (
	errorCodeLength   = 6
	errorStatusHeader = "error-status"
	errorCodeHeader   = "error-code"
	errorTextHeader   = "error-text"
)

var (
	// ErrorHeaderPrefix is the prefix to add to the http headers for ErrResponse objects
	//
	// The following headers are created:
	//    * ${ErrorHeaderPrefix}error-status
	//    * ${ErrorHeaderPrefix}error-code
	//    * ${ErrorHeaderPrefix}error-text
	//
	ErrorHeaderPrefix = "chi-render-"

	// GenErrorPin will generate a random 6 digit number that will be used to identify
	// the message in logs. Replace this if you want to change the way the error code
	// is generated
	GenErrorPin = func() string {
		var pin [errorCodeLength]byte
		// Don't care about the number of bytes read
		// Can only return oi.EOF or oi.UnexpectedEOF, which we don't care about
		_, _ = rand.Read(pin[:])
		for i := range pin {
			// grab the random number and modula it with 10 to get it to be from 0-9
			// turn it in ascii rep by adding '0' to it.
			pin[i] = (pin[i] % 10) + '0'
		}
		return string(pin[:])
	}
	// ErrorLogTo is the default error logging function.
	//
	// If you want all your ErrResponse based errors to log when
	// they are render be sure to set this variable. Once can easily
	// set it in an Init function:
	//
	// e.g.
	//
	//    func init(){
	//       render.ErrorHeaderPrefix = "my-prefix-"
	//       render.ErrorLogTo = render.ErrLogToStdOut
	//    }
	//
	ErrorLogTo func(*ErrResponse)
)

// ErrLogToStdOut can be used to use go log to log out the error when it is rendered
func ErrLogToStdOut(err *ErrResponse) {
	log.Printf("[StatusCode=%v %v] [ErrorCode=%v %v] [%+v]", err.StatusCode, err.StatusText, err.ErrorCode, err.ErrorText, err.Err)
}

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err        error  `json:"-"`               // low-level runtime error
	StatusCode int    `json:"-"`               // http response status code
	StatusText string `json:"status"`          // user-level status message
	ErrorCode  string `json:"code"`            // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
	// If you want to print out the issue set this the default ErrLogTo
	LogTo func(*ErrResponse) `json:"-"`
}

// Render will be called by the render to modify the ErrResponse object before it gets
// encoded by the Responders
func (err *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {

	// Generate a pseudo-unique error code
	err.ErrorCode = GenErrorPin()
	if err.StatusText == "" {
		err.StatusText = http.StatusText(err.StatusCode)
	}
	if err.ErrorText == "" {
		if err.Err == nil {
			err.ErrorText = err.StatusText
		} else {
			err.ErrorText = err.Err.Error()
		}
	}

	// Set the http response status based on the error
	Status(r, err.StatusCode)

	// Add the err response fields to the header, for clients that cannot parse the request body
	w.Header().Set(ErrorHeaderPrefix+errorStatusHeader, err.StatusText)
	w.Header().Set(ErrorHeaderPrefix+errorCodeHeader, err.ErrorCode)
	w.Header().Set(ErrorHeaderPrefix+errorTextHeader, err.ErrorText)

	// Log the application-level error info for debugging
	if err.LogTo != nil {
		err.LogTo(err)
	} else if ErrorLogTo != nil {
		ErrorLogTo(err)
	}

	return nil
}
