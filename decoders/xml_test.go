package decoders_test

import (
	"encoding/xml"
	"testing"

	"github.com/gdey/chi-render/decoders"
	"github.com/gdey/chi-render/decoders/test"
)

func TestXML(t *testing.T) {
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

	tests := map[string]test.Case{
		"person": test.NewStringCase(`<person id="13">
    <name>
        <first>John</first>
        <last>Doe</last>
    </name>
    <age>42</age>
    <Married>false</Married>
    <!-- Need more details. -->
</person>`,
			Person{
				XMLName:   xml.Name{Local: "person"},
				Id:        13,
				FirstName: "John",
				LastName:  "Doe",
				Age:       42,
				Comment:   " Need more details. ",
			},
		),
	}
	for name, tc := range tests {
		t.Run(name, tc.Test(decoders.XML))
	}
}
