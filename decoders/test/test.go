package test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gdey/chi-render/decoders"
)

// Case is a test case for a Decoder
type Case struct {
	// R is the input to decode
	R io.Reader

	// Value is the expected value of decoding R if Err is nil
	Value interface{}

	// Err is the expected Err for decoding R
	Err error

	// ErrComparator will be used if defined to compare the errors
	ErrComparator func(expected, got error) bool

	// ValueComparator will be used if defined to compare the values
	ValueComparator func(expected, got interface{}) bool
}

func defaultErrComparator(expected, got error) bool {
	if errors.Is(expected, got) {
		return true
	}
	vErr := reflect.ValueOf(expected).Interface()
	return errors.As(got, &vErr)
}

// Test will test the decoder to insure it is working correctly
func (tc Case) Test(decoder decoders.Func) func(*testing.T) {
	val := reflect.New(reflect.TypeOf(tc.Value)).Interface()
	if tc.ErrComparator == nil {
		tc.ErrComparator = defaultErrComparator
	}
	if tc.ValueComparator == nil {
		tc.ValueComparator = reflect.DeepEqual
	}
	return func(t *testing.T) {
		err := decoder(tc.R, val)
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
		elem := reflect.ValueOf(val).Elem().Interface()
		if !tc.ValueComparator(tc.Value, elem) {
			t.Errorf("value, expected %#v, got %#v", tc.Value, elem)
		}
	}
}

func NewStringCase(input string, value interface{}) Case {
	return Case{
		R:     strings.NewReader(input),
		Value: value,
	}
}
func NewStringErrCase(input string, err error) Case {
	return Case{
		R:   strings.NewReader(input),
		Err: err,
	}
}
func NewFileCase(filename string, value interface{}) Case {
	f, err := os.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open %s:%v", filename, err))
	}
	return Case{
		R:     f,
		Value: value,
	}
}
func NewFileErrCase(filename string, err error) Case {
	f, err := os.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open %s:%v", filename, err))
	}
	return Case{
		R:   f,
		Err: err,
	}
}
