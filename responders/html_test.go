package responders_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gdey/chi-render/responders"
	"github.com/gdey/chi-render/responders/helpers"
	"github.com/gdey/chi-render/responders/test"
)

type HTMLString string

func (str HTMLString) MarshalHTML() ([]byte, error) {
	var buff bytes.Buffer
	buff.WriteString(`<html><head><title>`)
	buff.WriteString(string(str))
	buff.WriteString(`</title></head><body><h1>`)
	buff.WriteString(string(str))
	buff.WriteString(`</h1></body></html>`)
	return buff.Bytes(), nil
}

func TestHTML(t *testing.T) {

	stdHeaders := func(tc *test.Case) *test.Case {
		if tc.R == nil {
			tc.R = new(http.Request)
			helpers.Status(tc.R, tc.W.Status)
		}
		if tc.W.Headers == nil {
			tc.W.Headers = make(http.Header)
		}
		helpers.SetNoSniffHeader(test.AsHeaderer(tc.W.Headers))
		helpers.SetContentTypeHeader(test.AsHeaderer(tc.W.Headers), "text/html; charset=utf-8")

		return tc
	}

	errMarshaller := errors.New("expected marshaller error")
	tests := map[string]test.Case{
		"string": func() test.Case {
			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader("Hello world!"),
				},
				V: "Hello world!",
			})
			return *tc
		}(),
		"TextMarshaler": func() test.Case {

			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader("2027-08-17T18:19:30.000004353Z"),
				},
				V: time.Date(2022, 67, 45, 89, 78, 90, 4353, time.UTC),
			})
			return *tc
		}(),
		"TextMarshaler Error": {
			V:   TextMarshalerError{errMarshaller},
			Err: errMarshaller,
		},
		"HTMLMarshaler Error": {
			V:   HTMLMarshalerError{errMarshaller},
			Err: errMarshaller,
		},
		"Stringer": func() test.Case {
			u, _ := url.Parse("https://example.org")
			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader("https://example.org"),
				},
				V: u,
			})
			return *tc
		}(),
		"HTMLMarshaler": func() test.Case {
			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader(`<html><head><title>hello world!</title></head><body><h1>hello world!</h1></body></html>`),
				},
				V: HTMLString("hello world!"),
			})
			return *tc
		}(),
		"ErrCanNotEncode": {
			Err: responders.ErrCanNotEncodeObject,
			V:   42,
		},
	}
	for name, tc := range tests {
		t.Run(name, tc.Test(responders.HTML))
	}
}
