package test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/gdey/chi-render/responders"
)

type AsHeaderer http.Header

func (h AsHeaderer) Header() http.Header { return http.Header(h) }

type ResponseWriter struct {

	// Status is the http status that should be set
	Status int

	// Headers that must exist and their values
	Headers http.Header

	// Body is the expected body
	Body io.Reader

	// StatusCodeComparator will be used to check the status code
	StatusCodeComparator func(expected, got int) bool

	// HeadersComparator will be used to check the headers
	HeaderComparator func(expected, got http.Header) bool

	// BodyComparator will be used to check the body
	BodyComparator func(expected, got []byte) bool

	// Where the value get written to
	headers http.Header
	body    bytes.Buffer
	status  int
}

func (mrw *ResponseWriter) Header() http.Header {
	if mrw.headers == nil {
		mrw.headers = make(http.Header)
	}
	return mrw.headers
}
func (mrw *ResponseWriter) Write(b []byte) (int, error) { return mrw.body.Write(b) }
func (mrw *ResponseWriter) WriteHeader(statusCode int)  { mrw.status = statusCode }

func defaultHeaderComparator(expected, got http.Header) bool {
	// we will loop through expected key and make sure they
	// exists and has all the values for that key.
	for name, values := range expected {
		gotValues, ok := got[name]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(values, gotValues) {
			return false
		}
	}
	return true
}

func (mrw *ResponseWriter) CheckHeaders(t *testing.T) bool {
	t.Helper()
	cmp := mrw.HeaderComparator
	if cmp == nil {
		cmp = defaultHeaderComparator
	}
	if !cmp(mrw.Headers, mrw.headers) {
		t.Errorf("headers, expected %v, got %v", mrw.Headers, mrw.headers)
		return false
	}
	return true
}
func (mrw *ResponseWriter) CheckStatusCode(t *testing.T) bool {
	t.Helper()
	cmp := mrw.StatusCodeComparator
	if cmp == nil {
		cmp = func(expected, got int) bool { return expected == got }
	}
	if !cmp(mrw.Status, mrw.status) {
		t.Errorf("StatusCode, expected %v, got %v", mrw.Status, mrw.status)
		return false
	}
	return true
}

func (mrw *ResponseWriter) CheckBody(t *testing.T) bool {
	t.Helper()
	cmp := mrw.BodyComparator
	expectedBytes, err := ioutil.ReadAll(mrw.Body)
	if err != nil {
		panic(fmt.Sprintf("could not read expected value for %s: %v", t.Name(), err))
	}

	gotBytes := mrw.body.Bytes()

	if cmp == nil {
		cmp = func(_, _ []byte) bool {
			return reflect.DeepEqual(expectedBytes, gotBytes)
		}
	}

	if !cmp(expectedBytes, gotBytes) {
		t.Errorf("bodies did not match")
		t.Logf("expected:\n`%s`", expectedBytes)
		t.Logf("got:\n`%s`", gotBytes)
		return false
	}
	return true
}

// Case is a test case for a responder
type Case struct {
	// V is the value to be encode and written to the Responder
	V interface{}

	// R is the http.Request object
	R *http.Request

	// W is the expected values that should be written
	W ResponseWriter

	// Err is the expected error
	Err error

	// ErrComparator will be used if defined to compare the errors
	ErrComparator func(expected, got error) bool
}

func defaultErrComparator(expected, got error) bool {
	if errors.Is(expected, got) {
		return true
	}
	vErr := reflect.ValueOf(expected).Interface()
	return errors.As(got, &vErr)
}

func (tc Case) Test(responder responders.Func) func(*testing.T) {
	if tc.ErrComparator == nil {
		tc.ErrComparator = defaultErrComparator
	}
	return func(t *testing.T) {
		err := responder(&tc.W, tc.R, tc.V)
		if tc.Err != nil {
			if !tc.ErrComparator(tc.Err, err) {
				// out error does not match.
				t.Errorf("error, expected %v, got %v", tc.Err, err)
			}
			return
		}
		if err != nil {
			t.Errorf("error, expected nil, got %v", err)
			return
		}
		if !tc.W.CheckBody(t) {
			return
		}
		if !tc.W.CheckHeaders(t) {
			return
		}
		if !tc.W.CheckStatusCode(t) {
			return
		}

	}
}
