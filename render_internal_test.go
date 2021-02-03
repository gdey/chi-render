package render

import (
	"net/http"
	"testing"

	"github.com/gdey/chi-render/responders/test"
)

func TestRender(t *testing.T) {
	type tcase struct {
		// V is the value to be encode and written to the Responder
		V Renderer

		// R is the http.Request object
		R *http.Request

		// W is the expected values that should be written
		W test.ResponseWriter

		// Err is the expected error
		Err error
	}

	fn := func(tc tcase) func(t *testing.T) {

		return func(t *testing.T) {
			if tc.R == nil {
				tc.R = new(http.Request)
			}

			err := renderer(&tc.W, tc.R, tc.V)
			if err != nil {
				t.Errorf("error, expected nil, got %v", err)
			}

		}
	}

	tests := map[string]tcase{
		"no panic with nilRender": {
			V: struct {
				NilRender
			}{},
		},
		"no panic with nilRender w private": {
			V: struct {
				NilRender
				foo int
			}{},
		},
		"no panic with nilRender w non render struct": {
			V: struct {
				NilRender
				foo struct {
					bar int
				}
			}{},
		},
		"no panic with nilRender w non render ptr struct": {
			V: struct {
				NilRender
				foo *struct {
					bar int
				}
			}{},
		},
		"no panic with nilRender w private nilRender": {
			V: struct {
				NilRender
				foo struct {
					NilRender
				}
			}{},
		},
		"no panic with nilRender w private ptr nilRender": {
			V: struct {
				NilRender
				foo struct {
					*NilRender
				}
			}{},
		},
		"no panic with ptr nilRender w private ptr nilRender": {
			V: &struct {
				NilRender
				foo struct {
					*NilRender
				}
			}{},
		},
		"no panic with ptr nilRender w private ptr ": {
			V: &struct {
				NilRender
				foo *struct {
					bar int
				}
			}{},
		},
	}
	for name, tc := range tests {
		t.Run(name, fn(tc))
	}
}
