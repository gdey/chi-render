# Decoders

This package provides various decoders to handle decoding of incoming REQUEST bodies.

The following decoders are provides out of the box:

  * [JSON](json.go) handles decoding json objects
  * [XML](xml.go) handles  decoding xml objects

# Writing and registering your own decoders

A decoder is simply a function that matches the `decoders.Func`
func. 

```go

func JSON(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	return json.NewDecoder(r).Decode(v)
}

```

To register the docoder with a `render.Controller`, all that is need
is to call the `SetDecoder` method with the given content-type the
decoder will handle.

```go

// This is using the default Controller
render.SetDecoder(ContentType("application/my-json"),JSON)

```

Some predefined content type can be found at [content_type.go](../content_type.go#L182)