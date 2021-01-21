package responders_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gdey/chi-render/responders/helpers"

	"github.com/gdey/chi-render/responders"
	"github.com/gdey/chi-render/responders/test"
)

func TestJSON(t *testing.T) {

	stdHeaders := func(tc *test.Case) *test.Case {
		if tc.R == nil {
			tc.R = new(http.Request)
			helpers.Status(tc.R, tc.W.Status)
		}
		if tc.W.Headers == nil {
			tc.W.Headers = make(http.Header)
		}
		helpers.SetNoSniffHeader(test.AsHeaderer(tc.W.Headers))
		helpers.SetContentTypeHeader(test.AsHeaderer(tc.W.Headers), "application/json; charset=utf-8")

		return tc
	}

	tests := map[string]test.Case{
		"empty": func() test.Case {

			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader("{}\n"), // json.Encoder always add a newline
				},
				V: make(map[string]interface{}),
			})
			return *tc
		}(),
		"hello world": func() test.Case {

			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body:   strings.NewReader("{\"greeting\":\"hello\",\"name\":\"world\"}\n"), // json.Encoder always add a newline
				},
				V: map[string]interface{}{"greeting": "hello", "name": "world"},
			})
			return *tc
		}(),
	}
	for name, tc := range tests {
		t.Run(name, tc.Test(responders.JSON))
	}
}
