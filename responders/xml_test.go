package responders_test

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"

	"github.com/gdey/chi-render/responders"
	"github.com/gdey/chi-render/responders/helpers"
	"github.com/gdey/chi-render/responders/test"
)

func TestXML(t *testing.T) {

	stdHeaders := func(tc *test.Case) *test.Case {
		if tc.R == nil {
			tc.R = new(http.Request)
			helpers.Status(tc.R, tc.W.Status)
		}
		if tc.W.Headers == nil {
			tc.W.Headers = make(http.Header)
		}
		helpers.SetNoSniffHeader(test.AsHeaderer(tc.W.Headers))
		helpers.SetContentTypeHeader(test.AsHeaderer(tc.W.Headers), "application/xml; charset=utf-8")

		return tc
	}

	tests := map[string]test.Case{
		"person": func() test.Case {
			type Person struct {
				XMLName   xml.Name `xml:"person"`
				Id        int      `xml:"id,attr"`
				FirstName string   `xml:"name>first"`
				LastName  string   `xml:"name>last"`
				Age       int      `xml:"age"`
				Height    float32  `xml:"height,omitempty"`
				Married   bool
				Comment   string `xml:",comment"`
			}
			tc := stdHeaders(&test.Case{
				W: test.ResponseWriter{
					Status: http.StatusOK,
					Body: strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<person id="13"><name><first>John</first><last>Doe</last></name><age>42</age><Married>false</Married><!-- Need more details. --></person>`), // json.Encoder always add a newline
				},
				V: Person{
					XMLName:   xml.Name{Local: "person"},
					Id:        13,
					FirstName: "John",
					LastName:  "Doe",
					Age:       42,
					Comment:   " Need more details. ",
				},
			})
			return *tc
		}(),
	}
	for name, tc := range tests {
		t.Run(name, tc.Test(responders.XML))
	}
}
