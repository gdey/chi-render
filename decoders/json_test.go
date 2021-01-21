package decoders_test

import (
	"testing"

	"github.com/gdey/chi-render/decoders"

	"github.com/gdey/chi-render/decoders/test"
)

func TestJSON(t *testing.T) {
	tests := map[string]test.Case{
		"first": test.NewStringCase(
			`{"name":"world"}`,
			struct {
				Name string `json:"name"`
			}{Name: "world"},
		),
	}
	for name, tc := range tests {
		t.Run(name, tc.Test(decoders.JSON))
	}
}
